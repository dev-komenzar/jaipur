package converter

import (
	"fmt"
	"image/png"
	"os"
	"path/filepath"

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

// runMozjpeg converts a single image to JPEG using mozjpeg.
// inputPath is the full path to the source image.
// outputDir is the directory to write the converted file into.
// outputName is the predetermined output file name (including the .jpg extension).
func runMozjpeg(inputPath string, outputDir string, outputName string, q uint) error {
	outputPath := filepath.Join(outputDir, outputName)

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
