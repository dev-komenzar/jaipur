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

func removeWords(name string, removes []string) string {
	for _, r := range removes {
		name = strings.Replace(name, r, "", 1)
	}
	return name
}

func haveSomePrefix(name string) bool {
	// True means name has (ERROR) prefix
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
