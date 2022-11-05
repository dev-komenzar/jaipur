package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func check(e error) error {
	if e != nil {
		fmt.Println(os.Getwd())
		log.Fatal(e)
		return e
	}
	return nil
}

func changeDir(dir string) {
	whereAmI("read.go: 16: where am i: ")
	err := os.Chdir(dir)
	check(err)
	whereAmI("read.go: 17: where am i: ")
}

func whereAmI(comment string) {
	p, _ := os.Getwd()
	fmt.Println(comment, p)
}

func isFile(path string) bool {
	if f, _ := os.Stat(path); f.IsDir() {
		return false
	} else {
		return true
	}
}

// https://qiita.com/KemoKemo/items/d135ddc93e6f87008521
func getFileNameWithoutExt(path string) string {
	return filepath.Base(path[:len(path)-len(filepath.Ext(path))])
}

func removeWords(name string, removes []string) string {
	for _, r := range removes {
		name = strings.Replace(name, r, "", 1)
	}
	return name
}

// name arg should be file name like xxxx.zip
func rename(name string, removes []string) string {
	ext := filepath.Ext(name)
	name = getFileNameWithoutExt(name)
	name = removeWords(name, removes)
	name = strings.TrimSpace(name)
	return strings.Join([]string{name, ext}, "")
}

// True means name has (ERROR) prefix
func haveSomePrefix(name string) bool {
	reg := regexp.MustCompile(`^\(ERROR\)|^\(BIG\)`)
	return reg.MatchString(name)
}

func isExistInBox(name string) bool {
	if f, err := os.Stat(filepath.Join("box", name)); os.IsNotExist(err) || f.IsDir() {
		return false
	}
	return true
}

func addPrefix(name string, prefix string) (string, error) {
	newName := strings.Join([]string{prefix, name}, "")
	err := os.Rename(name, newName)
	return newName, err
}
