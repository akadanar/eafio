package imageutils

import (
	"bytes"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"math/rand/v2"
)

func CropRandomSquare(inputBuffer []byte) ([]byte, error) {
	img, format, err := image.Decode(bytes.NewReader(inputBuffer))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image: %v", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	maxCropSize := 680
	if width < height {
		if width < maxCropSize {
			maxCropSize = width
		}
	} else {
		if height < maxCropSize {
			maxCropSize = height
		}
	}

	minCropSize := 100
	if maxCropSize < minCropSize {
		return nil, fmt.Errorf("image too small to crop: min 200px required")
	}

	cropSize := rand.IntN(maxCropSize-minCropSize+1) + minCropSize
	maxX := width - cropSize
	maxY := height - cropSize

	randAxis := rand.IntN(2) // 0: X-priority, 1: Y-priority
	var startX, startY int

	if randAxis == 0 {
		if maxX > 0 {
			startX = rand.IntN(maxX + 1)
		}
		if maxY > 0 {
			startY = rand.IntN(maxY/2 + 1)
			if rand.IntN(2) == 0 {
				startY = maxY - startY
			}
		}
	} else {
		if maxY > 0 {
			startY = rand.IntN(maxY + 1)
		}
		if maxX > 0 {
			startX = rand.IntN(maxX/2 + 1)
			if rand.IntN(2) == 0 {
				startX = maxX - startX
			}
		}
	}

	rect := image.Rect(startX, startY, startX+cropSize, startY+cropSize)
	cropped := img.(interface {
		SubImage(r image.Rectangle) image.Image
	}).SubImage(rect)

	var outBuf bytes.Buffer
	switch format {
	case "png":
		err = png.Encode(&outBuf, cropped)
	case "jpeg", "jpg":
		err = jpeg.Encode(&outBuf, cropped, &jpeg.Options{Quality: 90})
	default:
		return nil, fmt.Errorf("unsupported image format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to encode cropped image: %v", err)
	}

	return outBuf.Bytes(), nil
}
