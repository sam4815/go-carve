package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"log"
	"math"
	"syscall/js"

	"image/draw"
	"image/jpeg"
	_ "image/png"
)

func calculateSeam(costs []float64, rows int, cols int) []int {
	seam := make([]int, rows)
	minStart := math.Inf(1)
	for x := 0; x < cols; x++ {
		if costs[x] < minStart {
			minStart = costs[x]
			seam[0] = x
		}
	}

	for y := 1; y < rows; y++ {
		x := seam[y-1]
		rowIdx := y * cols
		leftPixel, midPixel, rightPixel := x-1+rowIdx, x+rowIdx, x+1+rowIdx

		if leftPixel < 0 {
			leftPixel = 0
		}
		if rightPixel > cols-1 {
			rightPixel = cols - 1
		}

		leftPixelCost, midPixelCost, rightPixelCost := costs[leftPixel], costs[midPixel], costs[rightPixel]
		min := math.Min(math.Min(leftPixelCost, midPixelCost), rightPixelCost)

		if min == midPixelCost {
			seam[y] = x
		} else if min == rightPixelCost {
			seam[y] = x + 1
		} else {
			seam[y] = x - 1
		}
	}

	return seam
}

func removeSeam(costs *[]float64, img *image.RGBA, rows int, cols int, seam []int) {
	for y := 0; y < rows; y++ {
		skippedColIdx := seam[y]
		for x := skippedColIdx; x < cols-1; x++ {
			(*costs)[x+(y*cols)] = (*costs)[x+1+(y*cols)]
			pixel := img.At(x+1, y)
			img.Set(x, y, pixel)
		}
		(*costs)[cols-1+(y*cols)] = math.Inf(1)
	}
}

func drawImage(src *image.RGBA, rows int, cols int, pixels []float64) *image.RGBA {
	rect := image.Rect(0, 0, cols, rows)
	dst := image.NewRGBA(rect)

	for y := 0; y < rows; y++ {
		for x := 0; x < cols; x++ {
			pixel := src.At(x, y)
			dst.Set(x, y, pixel)
		}
	}

	return dst
}

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

	// Array to store edge values
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

	// Array to store cost values
	costImg := make([]float64, imgY*imgX+1)
	costMax := 0.0

	for y := imgY - 1; y >= 0; y-- {
		for x := 0; x < imgX; x++ {
			currentPixelCost := edgeDetectionImg[x+y*imgX]
			// Handle final row
			if y == imgY-1 {
				costImg[x+y*imgX] = currentPixelCost
				continue
			}

			rowIdx := (y + 1) * imgX
			leftPixelCost, midPixelCost, rightPixelCost := costImg[x-1+rowIdx], costImg[x+rowIdx], costImg[x+1+rowIdx]

			if x == 0 {
				leftPixelCost = math.Inf(1)
			} else if x == imgX-1 {
				rightPixelCost = math.Inf(1)
			}

			minCost := math.Min(math.Min(leftPixelCost, midPixelCost), rightPixelCost)

			cost := minCost + currentPixelCost
			costMax = math.Max(cost, costMax)

			costImg[x+imgX*y] = cost
		}
	}

	log.Println("Calculated costs")

	// Calculate and remove seams
	for i := 0; i < 200; i++ {
		seam := calculateSeam(costImg, imgY, imgX)
		removeSeam(&costImg, rgba, imgY, imgX, seam)
	}

	log.Println("Removed seams")

	final := drawImage(rgba, imgY, imgX-200, costImg)

	var buff bytes.Buffer
	jpeg.Encode(&buff, final, nil)

	return base64.StdEncoding.EncodeToString(buff.Bytes())
}

func main() {
	js.Global().Set("goCarve", js.FuncOf(carve))
	select {}
}