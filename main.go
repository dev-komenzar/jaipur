package main

import (
	"bufio"
	"fmt"
	"io"
	"log"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
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
				Usage:  "convert files. If arg is a directory, modify all files under the directory. Archives with subdirectories are flattened.",
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
	&cli.BoolFlag{
		Name:    "yes",
		Aliases: []string{"y"},
		Usage:   "Auto-confirm: flatten all archives with multiple subdirectories without prompting",
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

	// Pre-scan all files for multiple image subdirectories, then ask the user
	// (once) which ones to skip.
	skipSet, err := confirmSubdirs(cCtx, files, remove)
	if err != nil {
		return err
	}

	for i, f := range files {

		newname := rename(f.Name(), remove)

		if haveSomePrefix(f.Name()) || isExistInBox(newname) {
			fmt.Println("skipped: ", f.Name())
			continue
		}

		if skipSet[f.Name()] {
			addPrefix(f.Name(), "(SKIP) ")
			fmt.Println("skipped (user): ", f.Name())
			continue
		}

		converted, errconvert := converter.Convert(
			f.Name(),
			newname,
			cCtx.Uint("quality"),
		)

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

// confirmSubdirs pre-scans the files for archives whose images span multiple
// directories and returns the set of file names the user chose to skip.
func confirmSubdirs(cCtx *cli.Context, files []os.FileInfo, remove []string) (map[string]bool, error) {
	skip := map[string]bool{}

	if cCtx.Bool("yes") {
		return skip, nil
	}

	var flagged []os.FileInfo
	for _, f := range files {
		if haveSomePrefix(f.Name()) || isExistInBox(rename(f.Name(), remove)) {
			continue
		}
		n, err := converter.CountImageDirs(f.Name())
		if err != nil {
			// Can't inspect the archive; let the normal Convert error path
			// handle it rather than prompting.
			continue
		}
		if n >= 2 {
			flagged = append(flagged, f)
		}
	}

	if len(flagged) == 0 {
		return skip, nil
	}

	if !isTerminal(os.Stdin) {
		return nil, fmt.Errorf("confirmation required: %d file(s) have multiple subdirectories. Run interactively or pass --yes", len(flagged))
	}

	fmt.Println("以下のファイルは画像を含むサブディレクトリを複数含みます（フラット化されます）：")
	for i, f := range flagged {
		fmt.Printf("  [%d] %s\n", i+1, f.Name())
	}
	fmt.Print("スキップするファイルの番号をカンマ区切りで入力（例: 2 / 空Enter=全部処理 / \"all\"=全部スキップ）: ")

	reader := bufio.NewReader(os.Stdin)
	line, err := reader.ReadString('\n')
	if err != nil && err != io.EOF {
		return nil, err
	}

	idxs, err := parseSkipInput(strings.TrimSpace(line), len(flagged))
	if err != nil {
		return nil, err
	}
	for _, idx := range idxs {
		skip[flagged[idx].Name()] = true
	}
	return skip, nil
}

// parseSkipInput parses a comma-separated list of 1-based indices (or "all")
// into 0-based indices to skip. Empty input returns an empty list (skip none).
func parseSkipInput(input string, total int) ([]int, error) {
	input = strings.TrimSpace(input)
	if input == "" {
		return nil, nil
	}
	if strings.EqualFold(input, "all") {
		idxs := make([]int, total)
		for i := range idxs {
			idxs[i] = i
		}
		return idxs, nil
	}

	var idxs []int
	seen := map[int]struct{}{}
	for _, part := range strings.Split(input, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if strings.Contains(part, "-") {
			bounds := strings.SplitN(part, "-", 2)
			lo, err1 := strconv.Atoi(strings.TrimSpace(bounds[0]))
			hi, err2 := strconv.Atoi(strings.TrimSpace(bounds[1]))
			if err1 != nil || err2 != nil {
				return nil, fmt.Errorf("invalid range: %q", part)
			}
			if lo > hi {
				lo, hi = hi, lo
			}
			for n := lo; n <= hi; n++ {
				if err := addSkipIndex(&idxs, seen, n, total); err != nil {
					return nil, err
				}
			}
		} else {
			n, err := strconv.Atoi(part)
			if err != nil {
				return nil, fmt.Errorf("invalid index: %q", part)
			}
			if err := addSkipIndex(&idxs, seen, n, total); err != nil {
				return nil, err
			}
		}
	}
	sort.Ints(idxs)
	return idxs, nil
}

func addSkipIndex(idxs *[]int, seen map[int]struct{}, n, total int) error {
	if n < 1 || n > total {
		return fmt.Errorf("index out of range: %d (must be 1-%d)", n, total)
	}
	if _, ok := seen[n]; !ok {
		seen[n] = struct{}{}
		*idxs = append(*idxs, n-1)
	}
	return nil
}

// isTerminal reports whether f is attached to an interactive terminal.
func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return false
	}
	return fi.Mode()&os.ModeCharDevice != 0
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
	converted, err := converter.ConvertDirectory(
		absPath,
		outputName,
		cCtx.Uint("quality"),
		outputType,
	)
	if err != nil {
		return fmt.Errorf("conversion failed: %v", err)
	}

	fmt.Printf("Conversion complete: %s\n", converted)
	return nil
}
