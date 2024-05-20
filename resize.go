package kava

import (
	"bytes"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"path/filepath"

	"golang.org/x/image/bmp"
	"golang.org/x/image/draw"
	"golang.org/x/image/tiff"
)

type Quality uint

const (
	QualityLow Quality = iota + 1
	QualityMedium
	QualityHigh
	QualityVeryHigh
)

type ResizeOption struct {
	ImageToResize io.Reader
	ResizeWidth   int
	Quality       Quality
	Ext           string
}

func ResizeImage(opt *ResizeOption) ([]byte, error) {
	if opt == nil {
		return nil, fmt.Errorf("resize option is nil")
	}
	if opt.ImageToResize == nil {
		return nil, fmt.Errorf("no image reader to resize")
	}
	if opt.ResizeWidth == 0 {
		opt.ResizeWidth = 200
	}
	if opt.Quality == 0 {
		opt.Quality = QualityMedium
	}

	img, fname, err := image.Decode(opt.ImageToResize)
	if err != nil {
		return nil, fmt.Errorf("decode error: %v", err)
	}
	if opt.Ext == "" {
		opt.Ext = filepath.Ext(fname)
	}
	// Calculate new dimensions while preserving aspect ratio
	originalWidth, originalHeight := img.Bounds().Dx(), img.Bounds().Dy()

	aspectRatio := float64(originalWidth) / float64(originalHeight)
	newHeight := int(float64(opt.ResizeWidth) / aspectRatio)

	// Create a new image with the calculated dimensions
	resizedImg := image.NewRGBA(image.Rect(0, 0, opt.ResizeWidth, newHeight))

	switch opt.Quality {
	case QualityLow:
		draw.NearestNeighbor.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	case QualityMedium:
		draw.ApproxBiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	case QualityHigh:
		draw.BiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	case QualityVeryHigh:
		draw.CatmullRom.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	default:
		draw.ApproxBiLinear.Scale(resizedImg, resizedImg.Bounds(), img, img.Bounds(), draw.Over, nil)
	}

	// Encode the resized image back to JPEG
	var out bytes.Buffer
	switch opt.Ext {
	case ".png":
		err = png.Encode(&out, resizedImg)
	case ".jpg", ".jpeg":
		err = jpeg.Encode(&out, resizedImg, nil)
	case ".gif":
		err = gif.Encode(&out, resizedImg, nil)
	case ".bmp":
		err = bmp.Encode(&out, resizedImg)
	case ".tiff", ".tif":
		err = tiff.Encode(&out, resizedImg, nil)
	default:
		return nil, fmt.Errorf("encode error: Unsupported image type %s", opt.Ext)
	}
	if err != nil {
		return nil, fmt.Errorf("encode error: %v", err)
	}

	return out.Bytes(), nil
}
