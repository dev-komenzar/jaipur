package convert

import (
	"context"
	"fmt"
	"image"
	"log"
	"os"
	"path/filepath"

	"github.com/mholt/archiver/v3"
	"golang.org/x/sync/errgroup"
)

func check(e error) {
	if e != nil {
		fmt.Println(os.Getwd())
		log.Fatal(e)
	}
}

// dir arg should be unarchived directory such as '125342143'
// return slice of ONLY file name such as 'xxxxx.jpg'
func getImages(dir string) []string {

	files, err := os.ReadDir(dir)
	check(err)

	var imageNames []string
	for _, f := range files {
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

// Names arg should be archive file name
// q arg should be given via cli
// ./ - XXX.zip
//   - [unarchive dir]/
//   - [tmp dir]/
//
// return new file name with "box/", unarchived dir name,
// tmp dir name and error
func Convert(oldname string, newname string, q uint) (string, string, string, error) {

	fmt.Println("Converting ", newname)

	// Unarchive
	unarchived, err := os.MkdirTemp(".", "")
	if err != nil {
		return "", "", "", err
	}
	// defer os.RemoveAll(unarchived)

	err = archiver.Unarchive(oldname, unarchived)
	if err != nil {
		return "", "", "", err
	}

	images := getImages(unarchived)

	if !(len(images) > 10) {
		return "", "", "", fmt.Errorf("no images to convert. check file: %v", oldname)
	}

	// Convert images in unarchive dir
	tmp, err := os.MkdirTemp(".", "tmp")
	if err != nil {
		return "", "", "", err
	}
	// defer os.RemoveAll(tmp)

	eg, _ := errgroup.WithContext(context.Background())
	eg.SetLimit(5)

	for _, i := range images {
		name, unarchived, tmp, q := i, unarchived, tmp, q
		eg.Go(func() error {

			return worker(name, unarchived, tmp, q)

		})
	}

	if err := eg.Wait(); err != nil {
		return "", "", "", err
	}

	// Archive
	var targets []string
	for _, i := range getImages(tmp) {
		targets = append(targets, filepath.Join(tmp, i))
	}

	err = archiver.Archive(targets, "box/"+newname)
	check(err)

	return "box/" + newname, unarchived, tmp, nil
}
