package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v2"

	converter "jaipur/pkg/convert"
	"jaipur/pkg/read"
)

func main() {
	app := &cli.App{
		Flags: flags,
		Commands: []*cli.Command{
			{
				Name:   "convert",
				Usage:  "convert files. If arg is a directory, modify all files under the directory ",
				Action: convert,
				Flags:  flags,
			},
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

var flags []cli.Flag = []cli.Flag{
	&cli.StringSliceFlag{
		Name:  "remove",
		Usage: "A word you want to remove. ex '--remove a,b,c' ",
	},
	&cli.UintFlag{
		Name:    "quality",
		Aliases: []string{"q"},
		Value:   70,
		Usage:   "Set quality for mozjpeg",
	},
}

func convert(cCtx *cli.Context) error {

	fmt.Printf("Hello %q\n", cCtx.Args().Get(0))
	remove := cCtx.StringSlice("remove")
	fmt.Println("Remove word: ", remove)

	// Path isExist
	path := cCtx.Args().Get(0)
	fmt.Println(path)
	info, err := os.Stat(path)
	check(err)

	// Change directory
	var dir string
	if info.IsDir() {
		dir = path
	} else {
		dir = filepath.Dir(path)
	}
	fmt.Println(dir)
	changeDir(dir)

	// Argment should be dir or file.
	var files []os.FileInfo

	if f, _ := os.Stat(path); f.IsDir() {
		var err error
		files, err = read.ReadDir(path)
		if err != nil {
			return err
		}

	} else {

		f, err := os.Stat(path)
		if err != nil {
			return err
		}
		files = append(files, f)
	}

	// Check if "box" dir exits
	if f, err := os.Stat("box"); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir("box", 0777)
		if err != nil {
			return err
		}
	}
	// Check if "done" dir exits
	if f, err := os.Stat("done"); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir("done", 0777)
		if err != nil {
			return err
		}
	}

	for i, f := range files {

		newname := rename(f.Name(), cCtx.StringSlice("remove"))

		if haveSomePrefix(f.Name()) || isExistInBox(newname) {
			fmt.Println("skipped: ", f.Name())
			continue
		}

		// Current dir is:
		//  - XXX.zip
		//   - [unarchive dir]/
		//   - [tmp dir]/
		converted, unarchived, tmp, errconvert := converter.Convert(
			f.Name(),
			newname,
			cCtx.Uint("quality"),
		)
		fmt.Printf("Original images: %v, Converted: %v", unarchived, tmp)
		// Delete temporary directories only if arg is directory
		defer func() {
			if !(errconvert != nil && isFile(path)) {
				os.RemoveAll(unarchived)
				os.RemoveAll(tmp)
			}
		}()

		if errconvert != nil {
			log.Printf("error in %v \n", f.Name())
			log.Println(errconvert)
			// Add prefix to file name
			addPrefix(f.Name(), "(ERROR) ")
			continue
		}

		originalInfo := f
		convrtInfo, err := os.Stat(converted)
		if err != nil {
			return err
		}

		fmt.Printf("File %v Original: %vMB, Converted: %vMB, Converted/Original: %v percent\n",
			i,
			originalInfo.Size()/(1024*1024),
			convrtInfo.Size()/(1024*1024),
			math.Floor((float64(convrtInfo.Size())/float64(originalInfo.Size()))*100))

		// Add prefix BIG
		if convrtInfo.Size() > 250*1024*1024 {
			oldpath := converted
			newpath := strings.Join([]string{"box/", "(BIG) ", convrtInfo.Name()}, "")
			err = os.Rename(oldpath, newpath)
			if err != nil {
				return err
			}
		}

		// Remove file
		err = os.Rename(f.Name(), "done/"+f.Name())
		if err != nil {
			return err
		}
	}

	return nil
}
