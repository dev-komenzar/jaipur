package converter

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/gen2brain/avif"
	"github.com/mholt/archiver/v3"
	"golang.org/x/sync/errgroup"

	_ "image/png"
)

func check(e error) {
	if e != nil {
		fmt.Println(os.Getwd())
		log.Fatal(e)
	}
}

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

// dir arg should be unarchived directory such as '125342143'
//
// return slice of ONLY file name such as 'xxxxx.jpg'
func getImagesFromDir(dir string) []string {
	entries, err := os.ReadDir(dir)
	check(err)

	var imageNames []string
	for _, f := range entries {
		if f.IsDir() {
			continue
		}
		path := filepath.Join(dir, f.Name())

		// Try standard image formats first (PNG, JPEG, GIF)
		data, err := os.Open(path)
		if err != nil {
			continue
		}

		if _, _, err := image.DecodeConfig(data); err == nil {
			data.Close()
			imageNames = append(imageNames, f.Name())
			continue
		}
		data.Close()

		// Try AVIF
		if isAVIF(path) {
			imageNames = append(imageNames, f.Name())
		}
	}

	return imageNames
}

// Names arg should be archived file name WITHOUT path
// q arg should be given via cli
//
// return new file name with "box/", unarchived dir name,
// tmp dir name and error
func Convert(oldname string, newname string, q uint) (string, string, string, error) {

	fmt.Println("Converting ", newname)

	// Unarchive
	unarchivedDir, err := os.MkdirTemp(".", "")
	if err != nil {
		return "", "", "", err
	}
	// defer os.RemoveAll(unarchived)

	err = archiver.Unarchive(oldname, unarchivedDir)
	if err != nil {
		return "", "", "", err
	}

	images := getImagesFromDir(unarchivedDir)

	if !(len(images) > 10) {
		fmt.Println(images)
		return "", "", "", fmt.Errorf("no images to convert. check file: %v", oldname)
	}

	// Convert images in unarchive dir
	tmpDir, err := os.MkdirTemp(".", "tmp")
	if err != nil {
		return "", "", "", err
	}
	// defer os.RemoveAll(tmp)

	parallelism := max(1, runtime.NumCPU()/2)
	log.Printf("Converting %d images with %d parallel workers\n", len(images), parallelism)
	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(parallelism)

	progress := NewProgressCounter(len(images))

	for _, i := range images {
		name, unarchivedDir, tmpDir, q := i, unarchivedDir, tmpDir, q
		eg.Go(func() error {
			err := runMozjpeg(name, unarchivedDir, tmpDir, q)
			if err == nil {
				progress.Increment()
			}
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		progress.Done()
		return "", "", "", err
	}
	progress.Done()

	// Archive
	var targets []string
	for _, i := range getImagesFromDir(tmpDir) {
		targets = append(targets, filepath.Join(tmpDir, i))
	}

	err = archiver.Archive(targets, "box/"+newname)
	if err != nil {
		return "", "", "", err
	}

	return "box/" + newname, unarchivedDir, tmpDir, nil
}

// ConvertDirectory processes images in a directory directly (without unarchiving)
// dirPath: path to the image directory
// outputName: name for the output (without extension)
// q: quality for mozjpeg
// outputType: "zip" or "dir"
// Returns: output path, tmp dir path, error
func ConvertDirectory(dirPath string, outputName string, q uint, outputType string) (string, string, error) {
	fmt.Println("Converting directory:", dirPath)

	images := getImagesFromDir(dirPath)

	if len(images) == 0 {
		return "", "", fmt.Errorf("no images found in directory: %v", dirPath)
	}

	// Create temporary directory for converted images
	tmpDir, err := os.MkdirTemp(".", "tmp")
	if err != nil {
		return "", "", err
	}

	// Convert images in parallel
	parallelism := max(1, runtime.NumCPU()/2)
	log.Printf("Converting %d images with %d parallel workers\n", len(images), parallelism)
	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(parallelism)

	progress := NewProgressCounter(len(images))

	for _, i := range images {
		name, inputDir, tmpDir, q := i, dirPath, tmpDir, q
		eg.Go(func() error {
			err := runMozjpeg(name, inputDir, tmpDir, q)
			if err == nil {
				progress.Increment()
			}
			return err
		})
	}

	if err := eg.Wait(); err != nil {
		progress.Done()
		return "", "", err
	}
	progress.Done()

	var outputPath string

	if outputType == "dir" {
		// Output as directory
		outputPath = "box/" + outputName
		err = os.MkdirAll(outputPath, 0777)
		if err != nil {
			return "", tmpDir, err
		}

		// Move converted images to output directory
		convertedImages := getImagesFromDir(tmpDir)
		for _, img := range convertedImages {
			src := filepath.Join(tmpDir, img)
			dst := filepath.Join(outputPath, img)
			err = os.Rename(src, dst)
			if err != nil {
				return "", tmpDir, err
			}
		}
	} else {
		// Output as zip (default)
		var targets []string
		for _, i := range getImagesFromDir(tmpDir) {
			targets = append(targets, filepath.Join(tmpDir, i))
		}

		outputPath = "box/" + outputName + ".zip"
		err = archiver.Archive(targets, outputPath)
		if err != nil {
			return "", tmpDir, err
		}
	}

	return outputPath, tmpDir, nil
}
