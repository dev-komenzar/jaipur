package read

import (
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
)

func check(e error) {
	if e != nil {
		fmt.Println(os.Getwd())
		log.Fatal(e)
	}
}

func Read(dir string) []fs.DirEntry {
	files, err := os.ReadDir(dir)
	check(err)
	var filteredFiles []fs.DirEntry
	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext == ".zip" || ext == ".rar" {
			filteredFiles = append(filteredFiles, f)
		}
	}

	return filteredFiles
}
