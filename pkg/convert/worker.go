package convert

import (
	"path/filepath"

	"github.com/nickalie/go-mozjpegbin"
)

func worker(name string, input string, output string, q uint) error {
	// name arg should be image file name
	// input arg should be dir name
	// output arg should be dir name

	inputPath := filepath.Join(input, name)
	outputPath := filepath.Join(output, name)
	mozjpegbin.SkipDownload()
	err := mozjpegbin.NewCJpeg().Quality(q).
		InputFile(inputPath).
		OutputFile(outputPath).
		Run()
	return err
}
