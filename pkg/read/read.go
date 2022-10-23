package read

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
)

func whereAmI(comment string) {
	p, _ := os.Getwd()
	fmt.Println(comment, p)
}

func Read(dir string) []string {
	fmt.Println("read.go: 15: ", dir)

	whereAmI("read.go: 16: where am i: ")
	err := os.Chdir(dir)
	if err != nil {
		log.Fatal(err)
	}
	whereAmI("read.go: 17: where am i: ")

	files, err := os.ReadDir(dir)
	if err != nil {
		log.Fatal(err)
	}
	var filteredFiles []string
	for _, f := range files {
		ext := filepath.Ext(f.Name())
		if ext == ".zip" || ext == ".rar" {
			filteredFiles = append(filteredFiles, f.Name())
			fmt.Println(filteredFiles)
		}
	}

	return filteredFiles
}
