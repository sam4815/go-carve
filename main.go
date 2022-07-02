package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"log"
	"math"
	"syscall/js"

	"image/color"
	"image/draw"
	"image/jpeg"
	_ "image/png"
)

func carve(this js.Value, inputs []js.Value) interface{} {
	srcBytes, err := base64.StdEncoding.DecodeString(inputs[0].String())
	if err != nil {
		log.Println(err)
		return nil
	}

	reader := bytes.NewReader(srcBytes)
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Println(err)
		return nil
	} else {
		log.Println("Image loaded")
	}

	bounds := img.Bounds()

	rgba, ok := img.(*image.RGBA)
	if !ok {
		rgba = image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, bounds.Min, draw.Src)
	}

	imgX, imgY := bounds.Max.X, bounds.Max.Y
	paddedX, paddedY := bounds.Max.X+2, bounds.Max.Y+2
	paddedImg := make([]float64, paddedX*paddedY)

	// Calculate grayscale pixel and copy to padded image
	for y := 0; y < paddedY; y++ {
		for x := 0; x < paddedX; x++ {
			if x == 0 || y == 0 || x == bounds.Max.X+1 || y == bounds.Max.Y+1 {
				paddedImg[x+(paddedX*y)] = 0
				continue
			}

			r, g, b, _ := img.At(x-1, y-1).RGBA()
			grey := float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114
			pixel := uint8(grey / 256)
			paddedImg[x+(paddedX*y)] = float64(pixel)
		}
	}
	log.Println("Converted to grayscale")

	// Define Sobel filters
	sX := [9]float64{-1, -2, -1, 0, 0, 0, 1, 2, 1}
	sY := [9]float64{-1, 0, 1, -2, 0, 2, -1, 0, 1}

	// Result matrix
	edgeDetectionImg := make([]float64, imgY*imgX)
	edgeMax := 0.0

	// For each image pixel, perform element-wise multiplication
	// on the enclosing 3x3 pixels by the Sobel filters.
	for y := 1; y < imgY+1; y++ {
		for x := 1; x < imgX+1; x++ {
			xSum, ySum := 0.0, 0.0
			for z := 0; z < 9; z++ {
				pixelX, pixelY := x-1+z%3, y-1+z/3
				pixel := paddedImg[pixelX+paddedX*pixelY]

				xSum += sX[z] * pixel
				ySum += sY[z] * pixel
			}

			edge := math.Sqrt(math.Pow(ySum, 2) + math.Pow(xSum, 2))
			edgeMax = math.Max(edge, edgeMax)

			edgeDetectionImg[x-1+(y-1)*imgX] = edge
		}
	}

	log.Println("Calculated edges")

	// Convert the pixel array back to an image
	for y := 0; y < imgY; y++ {
		for x := 0; x < imgX; x++ {
			pixel := uint8((edgeDetectionImg[x+y*imgX] / edgeMax) * 255)
			rgba.Set(x, y, color.RGBA{pixel, pixel, pixel, 1})
		}
	}

	var buff bytes.Buffer
	jpeg.Encode(&buff, rgba, nil)

	return base64.StdEncoding.EncodeToString(buff.Bytes())
}

func main() {
	js.Global().Set("goCarve", js.FuncOf(carve))
	select {}
}
