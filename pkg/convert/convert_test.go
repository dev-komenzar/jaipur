package converter

import (
	"archive/zip"
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/gif"
	"image/jpeg"
	"image/png"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"testing"
)

func writePNG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := png.Encode(f, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
}

func writeJPEG(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err := jpeg.Encode(f, image.NewRGBA(image.Rect(0, 0, 1, 1)), &jpeg.Options{Quality: 90}); err != nil {
		t.Fatal(err)
	}
}

func writeGIF(t *testing.T, path string) {
	t.Helper()
	f, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	pal := color.Palette{color.Black, color.White}
	if err := gif.Encode(f, image.NewPaletted(image.Rect(0, 0, 1, 1), pal), nil); err != nil {
		t.Fatal(err)
	}
}

func createTree(t *testing.T, root string, relPaths []string) {
	t.Helper()
	for _, rel := range relPaths {
		p := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(p), 0755); err != nil {
			t.Fatal(err)
		}
		writePNG(t, p)
	}
}

func createZipBytes(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	w := zip.NewWriter(&buf)
	for name, content := range entries {
		f, err := w.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := f.Write(content); err != nil {
			t.Fatal(err)
		}
	}
	if err := w.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func writeZip(t *testing.T, path string, entries map[string][]byte) {
	t.Helper()
	if err := os.WriteFile(path, createZipBytes(t, entries), 0644); err != nil {
		t.Fatal(err)
	}
}

func chdir(t *testing.T, dir string) {
	t.Helper()
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
}

func mozjpegAvailable() bool {
	_, err := exec.LookPath("cjpeg")
	return err == nil
}

func zipEntryNames(t *testing.T, path string) []string {
	t.Helper()
	r, err := zip.OpenReader(path)
	if err != nil {
		t.Fatal(err)
	}
	defer r.Close()
	names := make([]string, 0, len(r.File))
	for _, f := range r.File {
		names = append(names, f.Name)
	}
	return names
}

func TestNaturalLess(t *testing.T) {
	tests := []struct {
		a, b string
		want bool
	}{
		{"img2", "img10", true},
		{"img10", "img2", false},
		{"img1", "img2", true},
		{"volume1", "volume2", true},
		{"volume2", "volume10", true},
		{"volume10", "volume2", false},
		{"a", "b", true},
		{"", "a", true},
		{"a", "", false},
		{"123", "124", true},
		{"img01", "img1", false}, // equal numeric value; img01 not less than img1
	}
	for _, tt := range tests {
		if got := naturalLess(tt.a, tt.b); got != tt.want {
			t.Errorf("naturalLess(%q, %q) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestPadWidth(t *testing.T) {
	tests := []struct {
		n, want int
	}{
		{1, 2}, {9, 2}, {99, 2}, {100, 3}, {101, 3}, {999, 3}, {1000, 4},
	}
	for _, tt := range tests {
		if got := padWidth(tt.n); got != tt.want {
			t.Errorf("padWidth(%d) = %d, want %d", tt.n, got, tt.want)
		}
	}
}

func TestIsImageFile(t *testing.T) {
	dir := t.TempDir()
	pngPath := filepath.Join(dir, "a.png")
	writePNG(t, pngPath)
	jpgPath := filepath.Join(dir, "b.jpg")
	writeJPEG(t, jpgPath)
	gifPath := filepath.Join(dir, "c.gif")
	writeGIF(t, gifPath)
	txtPath := filepath.Join(dir, "d.txt")
	if err := os.WriteFile(txtPath, []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}

	if !isImageFile(pngPath) {
		t.Error("png not detected")
	}
	if !isImageFile(jpgPath) {
		t.Error("jpeg not detected")
	}
	if !isImageFile(gifPath) {
		t.Error("gif not detected")
	}
	if isImageFile(txtPath) {
		t.Error("txt wrongly detected as image")
	}
}

func TestCollectImages(t *testing.T) {
	root := t.TempDir()
	createTree(t, root, []string{
		"img01.jpg",
		"volume1/img02.jpg",
		"volume1/img03.png",
	})
	// Non-image files should be ignored.
	if err := os.WriteFile(filepath.Join(root, "note.txt"), []byte("hello"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "volume1", "readme.txt"), []byte("hi"), 0644); err != nil {
		t.Fatal(err)
	}

	entries, err := collectImages(root)
	if err != nil {
		t.Fatal(err)
	}
	got := map[string]bool{}
	for _, e := range entries {
		got[e.relPath] = true
	}
	want := map[string]bool{
		"img01.jpg":         true,
		"volume1/img02.jpg": true,
		"volume1/img03.png": true,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("collectImages() = %v, want %v", got, want)
	}
}

func TestBuildJobs(t *testing.T) {
	tests := []struct {
		name     string
		relPaths []string
		want     []string
	}{
		{
			name:     "flat keeps original names",
			relPaths: []string{"img01.jpg", "img02.jpg"},
			want:     []string{"img01.jpg", "img02.jpg"},
		},
		{
			name:     "single wrapper keeps original names",
			relPaths: []string{"Best/img01.jpg", "Best/img02.jpg"},
			want:     []string{"img01.jpg", "img02.jpg"},
		},
		{
			name:     "multi subdir renumbers with prefix",
			relPaths: []string{"volume1/img01.jpg", "volume1/img02.jpg", "volume2/img01.jpg"},
			want:     []string{"volume1_img01.jpg", "volume1_img02.jpg", "volume2_img01.jpg"},
		},
		{
			name:     "single subdir keeps original names in natural order",
			relPaths: []string{"volume1/img10.jpg", "volume1/img2.jpg", "volume1/img1.jpg"},
			want:     []string{"img1.jpg", "img2.jpg", "img10.jpg"},
		},
		{
			name:     "mixed root keeps name subdir renumbers",
			relPaths: []string{"cover.jpg", "volume1/img01.jpg", "volume2/img02.jpg"},
			want:     []string{"cover.jpg", "volume1_img01.jpg", "volume2_img01.jpg"},
		},
		{
			name:     "subdirs in natural order",
			relPaths: []string{"volume2/img01.jpg", "volume10/img01.jpg", "volume1/img01.jpg"},
			want:     []string{"volume1_img01.jpg", "volume2_img01.jpg", "volume10_img01.jpg"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			createTree(t, root, tt.relPaths)
			jobs, err := buildJobs(root)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, len(jobs))
			for i, j := range jobs {
				got[i] = j.outputName
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("buildJobs() output names = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBuildJobsZeroPad(t *testing.T) {
	root := t.TempDir()
	rel := make([]string, 0, 101)
	for i := 1; i <= 100; i++ {
		rel = append(rel, fmt.Sprintf("volume1/img%03d.jpg", i))
	}
	rel = append(rel, "volume2/img01.jpg")
	createTree(t, root, rel)

	jobs, err := buildJobs(root)
	if err != nil {
		t.Fatal(err)
	}
	if len(jobs) != 101 {
		t.Fatalf("len(jobs) = %d, want 101", len(jobs))
	}
	if jobs[0].outputName != "volume1_img001.jpg" {
		t.Errorf("first = %q, want volume1_img001.jpg", jobs[0].outputName)
	}
	if jobs[99].outputName != "volume1_img100.jpg" {
		t.Errorf("100th = %q, want volume1_img100.jpg", jobs[99].outputName)
	}
	if jobs[100].outputName != "volume2_img01.jpg" {
		t.Errorf("last = %q, want volume2_img01.jpg", jobs[100].outputName)
	}
}

func TestBuildJobsEmpty(t *testing.T) {
	root := t.TempDir()
	jobs, err := buildJobs(root)
	if err != nil {
		t.Fatal(err)
	}
	if jobs != nil {
		t.Fatalf("buildJobs() on empty dir = %v, want nil", jobs)
	}
}

func TestCountImageDirs(t *testing.T) {
	tests := []struct {
		name    string
		entries map[string][]byte
		want    int
	}{
		{
			name: "flat",
			entries: map[string][]byte{
				"img01.jpg": []byte("x"),
				"img02.jpg": []byte("x"),
			},
			want: 1,
		},
		{
			name: "single wrapper",
			entries: map[string][]byte{
				"Best/img01.jpg": []byte("x"),
				"Best/img02.jpg": []byte("x"),
			},
			want: 1,
		},
		{
			name: "multi subdir",
			entries: map[string][]byte{
				"volume1/img01.jpg": []byte("x"),
				"volume2/img02.jpg": []byte("x"),
			},
			want: 2,
		},
		{
			name: "mixed root and subdir",
			entries: map[string][]byte{
				"cover.jpg":         []byte("x"),
				"volume1/img01.jpg": []byte("x"),
			},
			want: 2,
		},
		{
			name: "ignores non-image files",
			entries: map[string][]byte{
				"readme.txt": []byte("x"),
				"data.csv":   []byte("x"),
			},
			want: 0,
		},
		{
			name: "ignores macOS metadata",
			entries: map[string][]byte{
				"volume1/img01.jpg":    []byte("x"),
				"__MACOSX/._img01.jpg": []byte("x"),
				".DS_Store":            []byte("x"),
			},
			want: 1,
		},
		{
			name: "case insensitive extension",
			entries: map[string][]byte{
				"volume1/IMG01.JPG": []byte("x"),
				"volume2/img02.png": []byte("x"),
			},
			want: 2,
		},
		{
			name: "nested deep path counts once per leaf dir",
			entries: map[string][]byte{
				"a/b/img01.jpg": []byte("x"),
				"a/b/img02.jpg": []byte("x"),
			},
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			archive := filepath.Join(dir, "test.zip")
			writeZip(t, archive, tt.entries)
			got, err := CountImageDirs(archive)
			if err != nil {
				t.Fatal(err)
			}
			if got != tt.want {
				t.Fatalf("CountImageDirs() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestConvertEndToEnd(t *testing.T) {
	if !mozjpegAvailable() {
		t.Skip("cjpeg (mozjpeg) not available")
	}
	dir := t.TempDir()
	chdir(t, dir)

	src := filepath.Join(dir, "input.zip")
	writeZip(t, src, map[string][]byte{
		"volume1/img01.jpg": pngBytesForZip(t),
		"volume1/img02.jpg": pngBytesForZip(t),
		"volume2/img01.png": pngBytesForZip(t),
	})

	converted, err := Convert("input.zip", "input.zip", 70)
	if err != nil {
		t.Fatal(err)
	}
	if converted != "box/input.zip" {
		t.Fatalf("converted = %q, want box/input.zip", converted)
	}

	names := zipEntryNames(t, filepath.Join(dir, "box", "input.zip"))
	want := []string{"volume1_img01.jpg", "volume1_img02.jpg", "volume2_img01.jpg"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("output entries = %v, want %v", names, want)
	}
}

func TestConvertNoImages(t *testing.T) {
	dir := t.TempDir()
	chdir(t, dir)

	src := filepath.Join(dir, "empty.zip")
	writeZip(t, src, map[string][]byte{
		"readme.txt": []byte("no images here"),
	})

	if _, err := Convert("empty.zip", "empty.zip", 70); err == nil {
		t.Fatal("expected error for archive with no images")
	}
}

func TestConvertDirectoryEndToEnd(t *testing.T) {
	if !mozjpegAvailable() {
		t.Skip("cjpeg (mozjpeg) not available")
	}
	dir := t.TempDir()
	chdir(t, dir)

	imgDir := filepath.Join(dir, "images")
	createTree(t, imgDir, []string{"volume1/img01.jpg", "volume2/img02.jpg"})

	converted, err := ConvertDirectory(imgDir, "out", 70, "zip")
	if err != nil {
		t.Fatal(err)
	}
	if converted != "box/out.zip" {
		t.Fatalf("converted = %q, want box/out.zip", converted)
	}

	names := zipEntryNames(t, filepath.Join(dir, "box", "out.zip"))
	want := []string{"volume1_img01.jpg", "volume2_img01.jpg"}
	if !reflect.DeepEqual(names, want) {
		t.Fatalf("output entries = %v, want %v", names, want)
	}
}

func pngBytesForZip(t *testing.T) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := png.Encode(&buf, image.NewRGBA(image.Rect(0, 0, 1, 1))); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}
