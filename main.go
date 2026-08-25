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
		Usage: "A word you want to remove. ex: --remove 'a,b,c'",
	},
	&cli.UintFlag{
		Name:    "quality",
		Aliases: []string{"q"},
		Value:   70,
		Usage:   "Set quality for mozjpeg",
	},
	&cli.StringFlag{
		Name:    "directory",
		Aliases: []string{"d"},
		Usage:   "Path to image directory to process directly",
	},
	&cli.StringFlag{
		Name:    "output",
		Aliases: []string{"o"},
		Value:   "zip",
		Usage:   "Output format: 'zip' or 'dir'",
	},
}

func convert(cCtx *cli.Context) error {
	remove := cCtx.StringSlice("remove")
	directoryFlag := unescapeShellPath(cCtx.String("directory"))
	outputType := cCtx.String("output")

	// Validate output type
	if outputType != "zip" && outputType != "dir" {
		return fmt.Errorf("invalid output type: %s. Use 'zip' or 'dir'", outputType)
	}

	// Directory mode: process image directory directly
	if directoryFlag != "" {
		return convertDirectory(cCtx, directoryFlag, outputType)
	}

	// Archive mode (original behavior)
	fmt.Printf("Hello %q\n", cCtx.Args().Get(0))
	fmt.Println("Remove word: ", remove)

	// Path isExist
	// Unescape shell-style backslash sequences (e.g., "\ " -> " ")
	path := unescapeShellPath(cCtx.Args().Get(0))
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

	// After changeDir, we are inside the target directory.
	// Use "." to refer to current directory instead of the original (potentially relative) path.
	var files []os.FileInfo

	if info.IsDir() {
		var err error
		files, err = read.ReadDir(".")
		if err != nil {
			return err
		}
	} else {
		// Single file case: use the file name relative to current dir
		files = append(files, info)
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

// convertDirectory handles the --directory flag mode
func convertDirectory(cCtx *cli.Context, dirPath string, outputType string) error {
	// Validate: check if any args are archive files
	for i := 0; i < cCtx.Args().Len(); i++ {
		arg := cCtx.Args().Get(i)
		ext := strings.ToLower(filepath.Ext(arg))
		if ext == ".zip" || ext == ".rar" {
			return fmt.Errorf("--directory flag cannot be used with archive files. Remove --directory or provide a directory path")
		}
	}

	// Validate directory exists
	info, err := os.Stat(dirPath)
	if err != nil {
		return fmt.Errorf("directory not found: %s", dirPath)
	}
	if !info.IsDir() {
		return fmt.Errorf("path is not a directory: %s", dirPath)
	}

	// Get absolute path for proper handling
	absPath, err := filepath.Abs(dirPath)
	if err != nil {
		return err
	}

	// Use directory name as output name
	outputName := filepath.Base(absPath)
	if remove := cCtx.StringSlice("remove"); len(remove) > 0 {
		outputName = removeWords(outputName, remove)
		outputName = strings.TrimSpace(outputName)
	}

	// Ensure box directory exists
	if f, err := os.Stat("box"); os.IsNotExist(err) || !f.IsDir() {
		err = os.Mkdir("box", 0777)
		if err != nil {
			return err
		}
	}

	// Convert the directory
	converted, tmpDir, err := converter.ConvertDirectory(
		absPath,
		outputName,
		cCtx.Uint("quality"),
		outputType,
	)

	// Clean up temp directory
	defer func() {
		if tmpDir != "" {
			os.RemoveAll(tmpDir)
		}
	}()

	if err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	fmt.Printf("Conversion complete: %s\n", converted)
	return nil
}
