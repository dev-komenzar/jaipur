package convert

import (
	"fmt"
	"path/filepath"

	"github.com/nickalie/go-mozjpegbin"
)

// name arg should be image file name
// input arg should be dir name
// output arg should be dir name
func worker(name string, input string, output string, q uint) error {

	inputPath := filepath.Join(input, name)
	outputPath := filepath.Join(output, name)
	mozjpegbin.SkipDownload()
	err := mozjpegbin.NewCJpeg().Quality(q).
		InputFile(inputPath).
		OutputFile(outputPath).
		Run()
	if err != nil {
		return fmt.Errorf("processing %v: %v", inputPath, err)
	}
	return nil
}
