package converter

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

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

// dir arg should be unarchived directory such as '125342143'
//
// return slice of ONLY file name such as 'xxxxx.jpg'
func getImagesFromDir(dir string) []string {

	entries, err := os.ReadDir(dir)
	check(err)

	var imageNames []string
	for _, f := range entries {
		path := filepath.Join(dir, f.Name())
		data, err := os.Open(path)
		check(err)
		defer data.Close()

		if _, _, err := image.DecodeConfig(data); err == nil {
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

	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(5)

	for _, i := range images {
		name, unarchivedDir, tmpDir, q := i, unarchivedDir, tmpDir, q
		eg.Go(func() error {

			return runMozjpeg(name, unarchivedDir, tmpDir, q)

		})
	}

	if err := eg.Wait(); err != nil {
		return "", "", "", err
	}

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
