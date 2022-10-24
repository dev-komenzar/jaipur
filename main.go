package main

import (
	"fmt"
	"log"
	"math"
	"os"
	"strings"

	"github.com/urfave/cli/v2"

	"jaipur/pkg/convert"
	"jaipur/pkg/read"
)

func main() {
	app := &cli.App{
		Flags: []cli.Flag{
			&cli.StringSliceFlag{Name: "remove", Usage: "A word you want to remove. ex '--remove a,b,c' "},
			&cli.UintFlag{Name: "quality", Aliases: []string{"q"}, Value: 70, Usage: "Set quality for mozjpeg"},
		},
		Action: action,
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func action(cCtx *cli.Context) error {

	fmt.Printf("Hello %q\n", cCtx.Args().Get(0))
	remove := cCtx.StringSlice("remove")
	fmt.Println("Remove word: ", remove)

	dir := cCtx.Args().Get(0)
	changeDir(dir)
	fileNames := read.Read(dir)

	// Check if "box" dir exits
	if f, err := os.Stat("box"); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir("box", 0777)
		check(err)
	}
	// Check if "done" dir exits
	if f, err := os.Stat("done"); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir("done", 0777)
		check(err)
	}

	for i, f := range fileNames {

		newname := removeWords(f.Name(), cCtx.StringSlice("remove"))

		if haveSomePrefix(f.Name()) || isExistInBox(newname) {
			fmt.Println("skipped: ", f.Name())
			continue
		}

		converted, err := convert.Convert(
			f.Name(),
			newname,
			cCtx.Uint("quality"))

		if err != nil {
			log.Printf("error in %v \n", f.Name())
			log.Println(err)
			// Add prefix to file name
			addPrefix(f.Name(), "(ERROR) ")
			continue
		}

		originalInfo, _ := f.Info()
		convrtInfo, err := os.Stat(converted)
		check(err)

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
			check(err)
		}

		// Remove file
		err = os.Rename(f.Name(), "done/"+f.Name())
		check(err)
	}

	return nil
}
