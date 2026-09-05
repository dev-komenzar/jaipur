package converter

import (
	"context"
	"fmt"
	"image"
	"io/fs"
	"log"
	"os"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"

	"github.com/gen2brain/avif"
	"github.com/klauspost/compress/zip"
	"github.com/mholt/archiver/v3"
	"github.com/nwaples/rardecode"
	"golang.org/x/sync/errgroup"

	// Register image decoders so image.DecodeConfig can detect them.
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// isAVIF checks if the file is an AVIF image
func isAVIF(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	_, err = avif.DecodeConfig(f)
	return err == nil
}

// isImageFile reports whether path is a recognized image file by decoding its
// header (PNG/JPEG/GIF via image.DecodeConfig, AVIF via isAVIF).
func isImageFile(path string) bool {
	f, err := os.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

	if _, _, err := image.DecodeConfig(f); err == nil {
		return true
	}
	return isAVIF(path)
}

// isMetadataPath reports whether an archive entry is a metadata file that
// should never be treated as an image (macOS/Windows artifacts).
func isMetadataPath(name string) bool {
	base := path.Base(name)
	if base == ".DS_Store" || base == "Thumbs.db" || base == "desktop.ini" || strings.HasPrefix(base, "._") {
		return true
	}
	for _, part := range strings.Split(path.Clean(name), "/") {
		if part == "__MACOSX" {
			return true
		}
	}
	return false
}

// imageEntry is a single image file found under a directory tree.
type imageEntry struct {
	relPath string // path relative to the tree root, e.g. "volume1/img01.jpg"
}

// collectImages recursively collects every image file under dir, returning
// their paths relative to dir.
func collectImages(dir string) ([]imageEntry, error) {
	var entries []imageEntry
	err := filepath.WalkDir(dir, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(dir, p)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)
		if isMetadataPath(rel) {
			return nil
		}
		if !isImageFile(p) {
			return nil
		}
		entries = append(entries, imageEntry{relPath: rel})
		return nil
	})
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// isASCIIDigit reports whether c is a decimal digit.
func isASCIIDigit(c byte) bool { return c >= '0' && c <= '9' }

// naturalLess reports whether a should sort before b using natural
// (numeric-aware) ordering: "img2" sorts before "img10".
func naturalLess(a, b string) bool {
	for {
		if a == "" || b == "" {
			return len(a) < len(b)
		}
		ca, cb := a[0], b[0]
		da, db := isASCIIDigit(ca), isASCIIDigit(cb)
		switch {
		case da && db:
			na, nb := 0, 0
			for len(a) > 0 && isASCIIDigit(a[0]) {
				na = na*10 + int(a[0]-'0')
				a = a[1:]
			}
			for len(b) > 0 && isASCIIDigit(b[0]) {
				nb = nb*10 + int(b[0]-'0')
				b = b[1:]
			}
			if na != nb {
				return na < nb
			}
		case da != db:
			return da // digits sort before non-digits
		default:
			if ca != cb {
				return ca < cb
			}
			a, b = a[1:], b[1:]
		}
	}
}

// pathDir returns the directory part of a slash-separated relative path,
// returning "" for the root (files with no directory component).
func pathDir(rel string) string {
	d := path.Dir(rel)
	if d == "." {
		return ""
	}
	return d
}

// toJPGName converts a relative image path to a .jpg output base name.
func toJPGName(rel string) string {
	base := path.Base(rel)
	return strings.TrimSuffix(base, path.Ext(base)) + ".jpg"
}

// sanitizeDirName flattens a directory path into a filename-safe prefix.
func sanitizeDirName(d string) string {
	return strings.ReplaceAll(d, "/", "_")
}

// padWidth returns the zero-padding width for a sequence of n items.
func padWidth(n int) int {
	if n <= 99 {
		return 2
	}
	return len(strconv.Itoa(n))
}

// dedupeOutputNames mutates jobs in place so that every outputName is unique,
// appending "_2", "_3", ... to later duplicates.
func dedupeOutputNames(jobs []job) {
	seen := map[string]int{}
	for i := range jobs {
		name := jobs[i].outputName
		if c, ok := seen[name]; ok {
			ext := path.Ext(name)
			base := strings.TrimSuffix(name, ext)
			jobs[i].outputName = fmt.Sprintf("%s_%d%s", base, c+1, ext)
			seen[name] = c + 1
		} else {
			seen[name] = 1
		}
	}
}

// job represents a single image conversion: an input file and its
// predetermined output name (base name only, always .jpg).
type job struct {
	inputPath  string
	outputName string
}

// buildJobs collects all images under rootDir and produces the ordered list
// of conversion jobs, applying the flattening/naming rules:
//
//   - If all images live in a single directory (the root, or one wrapper
//     folder), original file names are kept.
//   - If images span multiple directories (>= 2), images in subdirectories are
//     renamed to "<subdir>_imgNN.jpg" (zero-padded, natural order) while
//     top-level images keep their original names.
func buildJobs(rootDir string) ([]job, error) {
	entries, err := collectImages(rootDir)
	if err != nil {
		return nil, err
	}
	if len(entries) == 0 {
		return nil, nil
	}

	// Group image paths by the directory they live in ("" == root).
	dirs := map[string][]string{}
	for _, e := range entries {
		d := pathDir(e.relPath)
		dirs[d] = append(dirs[d], e.relPath)
	}
	for d := range dirs {
		sort.SliceStable(dirs[d], func(i, j int) bool {
			return naturalLess(path.Base(dirs[d][i]), path.Base(dirs[d][j]))
		})
	}

	rootHasImages := len(dirs[""]) > 0

	subdirNames := make([]string, 0, len(dirs))
	for d := range dirs {
		if d != "" {
			subdirNames = append(subdirNames, d)
		}
	}
	sort.SliceStable(subdirNames, func(i, j int) bool {
		return naturalLess(subdirNames[i], subdirNames[j])
	})

	numDirs := len(subdirNames)
	if rootHasImages {
		numDirs++
	}
	multiDir := numDirs >= 2

	var jobs []job

	// Top-level images first, always keeping original names.
	for _, rel := range dirs[""] {
		jobs = append(jobs, job{
			inputPath:  filepath.Join(rootDir, filepath.FromSlash(rel)),
			outputName: toJPGName(rel),
		})
	}

	// Then subdirectories in natural order.
	for _, d := range subdirNames {
		files := dirs[d]
		width := padWidth(len(files))
		for idx, rel := range files {
			out := toJPGName(rel)
			if multiDir {
				out = fmt.Sprintf("%s_img%0*d.jpg", sanitizeDirName(d), width, idx+1)
			}
			jobs = append(jobs, job{
				inputPath:  filepath.Join(rootDir, filepath.FromSlash(rel)),
				outputName: out,
			})
		}
	}

	dedupeOutputNames(jobs)

	return jobs, nil
}

// imageExts is the set of file extensions recognized as images for the
// pre-scan (extension-only) detection.
var imageExts = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".png":  {},
	".gif":  {},
	".avif": {},
}

func hasImageExt(name string) bool {
	_, ok := imageExts[strings.ToLower(path.Ext(name))]
	return ok
}

// fileFullPath returns the full path of an archive entry, handling the
// format-specific Header types used by archiver v3 (ZIP vs RAR).
func fileFullPath(f archiver.File) string {
	switch h := f.Header.(type) {
	case zip.FileHeader:
		return h.Name
	case *rardecode.FileHeader:
		return h.Name
	default:
		return f.Name()
	}
}

// CountImageDirs returns the number of distinct directories (including the
// archive root) that contain at least one image file, based on file name
// extensions. It inspects the archive without extracting it.
func CountImageDirs(archive string) (int, error) {
	dirSet := map[string]struct{}{}
	err := archiver.Walk(archive, func(f archiver.File) error {
		if f.IsDir() {
			return nil
		}
		name := fileFullPath(f)
		if isMetadataPath(name) {
			return nil
		}
		if !hasImageExt(name) {
			return nil
		}
		d := pathDir(name)
		dirSet[d] = struct{}{}
		return nil
	})
	if err != nil {
		return 0, err
	}
	return len(dirSet), nil
}

// Names arg should be archived file name WITHOUT path
// q arg should be given via cli
//
// return new file name with "box/", and error
func Convert(oldname string, newname string, q uint) (string, error) {
	fmt.Println("Converting ", newname)

	// Unarchive
	unarchivedDir, err := os.MkdirTemp(".", "")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(unarchivedDir)

	err = archiver.Unarchive(oldname, unarchivedDir)
	if err != nil {
		return "", err
	}

	jobs, err := buildJobs(unarchivedDir)
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("no images to convert. check file: %v", oldname)
	}

	tmpDir, err := os.MkdirTemp(".", "tmp")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	parallelism := max(1, runtime.NumCPU()/2)
	log.Printf("Converting %d images with %d parallel workers\n", len(jobs), parallelism)
	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(parallelism)

	progress := NewProgressCounter(len(jobs))

	for _, j := range jobs {
		j := j
		eg.Go(func() error {
			err := runMozjpeg(j.inputPath, tmpDir, j.outputName, q)
			if err == nil {
				progress.Increment()
			}
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		progress.Done()
		return "", err
	}
	progress.Done()

	// Archive the converted files in the same order as jobs.
	targets := make([]string, len(jobs))
	for i, j := range jobs {
		targets[i] = filepath.Join(tmpDir, j.outputName)
	}

	err = archiver.Archive(targets, "box/"+newname)
	if err != nil {
		return "", err
	}

	return "box/" + newname, nil
}

// ConvertDirectory processes images in a directory directly (without
// unarchiving), applying the same flattening/naming rules as Convert.
// dirPath: path to the image directory
// outputName: name for the output (without extension)
// q: quality for mozjpeg
// outputType: "zip" or "dir"
// Returns: output path and error
func ConvertDirectory(dirPath string, outputName string, q uint, outputType string) (string, error) {
	fmt.Println("Converting directory:", dirPath)

	jobs, err := buildJobs(dirPath)
	if err != nil {
		return "", err
	}
	if len(jobs) == 0 {
		return "", fmt.Errorf("no images found in directory: %v", dirPath)
	}

	tmpDir, err := os.MkdirTemp(".", "tmp")
	if err != nil {
		return "", err
	}
	defer os.RemoveAll(tmpDir)

	parallelism := max(1, runtime.NumCPU()/2)
	log.Printf("Converting %d images with %d parallel workers\n", len(jobs), parallelism)
	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(parallelism)

	progress := NewProgressCounter(len(jobs))

	for _, j := range jobs {
		j := j
		eg.Go(func() error {
			err := runMozjpeg(j.inputPath, tmpDir, j.outputName, q)
			if err == nil {
				progress.Increment()
			}
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		progress.Done()
		return "", err
	}
	progress.Done()

	var outputPath string

	if outputType == "dir" {
		outputPath = "box/" + outputName
		err = os.MkdirAll(outputPath, 0777)
		if err != nil {
			return "", err
		}
		for _, j := range jobs {
			src := filepath.Join(tmpDir, j.outputName)
			dst := filepath.Join(outputPath, j.outputName)
			if err := os.Rename(src, dst); err != nil {
				return "", err
			}
		}
	} else {
		targets := make([]string, len(jobs))
		for i, j := range jobs {
			targets[i] = filepath.Join(tmpDir, j.outputName)
		}

		outputPath = "box/" + outputName + ".zip"
		err = archiver.Archive(targets, outputPath)
		if err != nil {
			return "", err
		}
	}

	return outputPath, nil
}
