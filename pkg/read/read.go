package read

import (
	"io/fs"
	"os"
	"path/filepath"
)

func ReadDir(dir string) ([]fs.FileInfo, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var filteredFiles []fs.FileInfo
	for _, f := range entries {
		ext := filepath.Ext(f.Name())
		if ext == ".zip" || ext == ".rar" {

			info, err := f.Info()
			if err != nil {
				return nil, err
			}

			filteredFiles = append(filteredFiles, info)
		}
	}

	return filteredFiles, nil
}
