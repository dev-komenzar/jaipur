package converter

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"
	"strings"

	"github.com/gen2brain/avif"
	"github.com/nickalie/go-mozjpegbin"
)

// convertAVIFtoPNG converts AVIF to PNG for mozjpeg processing
func convertAVIFtoPNG(inputPath string) (string, error) {
	f, err := os.Open(inputPath)
	if err != nil {
		return "", err
	}
	defer f.Close()

	img, err := avif.Decode(f)
	if err != nil {
		return "", err
	}

	// Create temp PNG file
	tmpFile, err := os.CreateTemp("", "avif-*.png")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	if err := png.Encode(tmpFile, img); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// name arg should be image file name
// input arg should be dir name
// output arg should be dir name
func runMozjpeg(name string, input string, output string, q uint) error {
	inputPath := filepath.Join(input, name)

	// 出力ファイル名の拡張子を .jpg に変更
	outputName := strings.TrimSuffix(name, filepath.Ext(name)) + ".jpg"
	outputPath := filepath.Join(output, outputName)

	// Check if AVIF and convert to PNG first
	actualInput := inputPath
	if isAVIF(inputPath) {
		pngPath, err := convertAVIFtoPNG(inputPath)
		if err != nil {
			return fmt.Errorf("converting AVIF %v: %v", inputPath, err)
		}
		defer os.Remove(pngPath)
		actualInput = pngPath
	}

	mozjpegbin.SkipDownload()
	err := mozjpegbin.NewCJpeg().Quality(q).
		InputFile(actualInput).
		OutputFile(outputPath).
		Run()
	if err != nil {
		return fmt.Errorf("processing %v: %v", inputPath, err)
	}
	return nil
}
