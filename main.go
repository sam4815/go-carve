package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"log"
	"math"
	"syscall/js"

	"image/color"
	"image/jpeg"
	_ "image/png"
)

type matrix[T any] struct {
	vals     []T
	rows     int
	cols     int
	initCols int
}

func createMatrix[T any](rows int, cols int) matrix[T] {
	matrix := matrix[T]{rows: rows, cols: cols, initCols: cols}
	matrix.vals = make([]T, rows*cols)
	return matrix
}

func (matrix matrix[T]) At(x int, y int) T {
	return matrix.vals[x+(y*matrix.initCols)]
}

func (matrix matrix[T]) Set(x int, y int, val T) {
	matrix.vals[x+(y*matrix.initCols)] = val
}

func (matrix matrix[T]) ForEach(lambda func(x int, y int)) {
	for y := 0; y < matrix.rows; y++ {
		for x := 0; x < matrix.cols; x++ {
			lambda(x, y)
		}
	}
}

func generatePaddedGrayscale(src matrix[color.Color]) matrix[float64] {
	paddedX, paddedY := src.cols+2, src.rows+2
	paddedMatrix := createMatrix[float64](paddedY, paddedX)

	paddedMatrix.ForEach(func(x int, y int) {
		if x == 0 || y == 0 || x == paddedX-1 || y == paddedY-1 {
			paddedMatrix.Set(x, y, 0)
			return
		}

		r, g, b, _ := src.At(x-1, y-1).RGBA()
		grayPixel := uint8((float64(r)*0.299 + float64(g)*0.587 + float64(b)*0.114) / 256)

		paddedMatrix.Set(x, y, float64(grayPixel))
	})

	return paddedMatrix
}

func calculateEdgeCosts(src matrix[float64]) matrix[float64] {
	// Define Sobel filters
	sX := [9]float64{-1, -2, -1, 0, 0, 0, 1, 2, 1}
	sY := [9]float64{-1, 0, 1, -2, 0, 2, -1, 0, 1}

	targetRows, targetCols := src.rows-2, src.cols-2
	edgeMatrix := createMatrix[float64](targetRows, targetCols)

	// For each image pixel, perform element-wise multiplication
	// on the enclosing 3x3 pixels by the Sobel filters.
	for y := 1; y < src.rows-1; y++ {
		for x := 1; x < src.cols-1; x++ {
			xSum, ySum := 0.0, 0.0
			for z := 0; z < 9; z++ {
				pixelX, pixelY := x-1+z%3, y-1+z/3
				pixel := src.At(pixelX, pixelY)

				xSum += sX[z] * pixel
				ySum += sY[z] * pixel
			}

			edge := math.Sqrt(math.Pow(ySum, 2) + math.Pow(xSum, 2))

			edgeMatrix.Set(x-1, y-1, edge)
		}
	}

	return edgeMatrix
}

func calculateCostPaths(costs matrix[float64]) matrix[float64] {
	costPaths := createMatrix[float64](costs.rows, costs.cols)

	for y := costs.rows - 1; y >= 0; y-- {
		for x := 0; x < costs.cols; x++ {
			currentPixelCost := costs.At(x, y)
			// Handle final row
			if y == costs.rows-1 {
				costPaths.Set(x, y, currentPixelCost)
				continue
			}

			leftPixelIndex, midPixelIndex, rightPixelIndex := x-1, x, x+1
			if x == 0 {
				leftPixelIndex = 0
			} else if x == costs.cols-1 {
				rightPixelIndex = costs.cols - 1
			}

			minCost := math.Min(math.Min(costPaths.At(leftPixelIndex, y+1), costPaths.At(midPixelIndex, y+1)), costPaths.At(rightPixelIndex, y+1))
			cost := minCost + currentPixelCost

			costPaths.Set(x, y, cost)
		}
	}

	return costPaths
}

func calculateSeam(src matrix[float64]) []int {
	seam := make([]int, src.rows)
	minStart := math.Inf(1)
	for x := 0; x < src.cols; x++ {
		if src.At(x, 0) < minStart {
			minStart = src.At(x, 0)
			seam[0] = x
		}
	}

	for y := 1; y < src.rows; y++ {
		x := seam[y-1]
		leftPixel, midPixel, rightPixel := x-1, x, x+1

		if leftPixel < 0 {
			leftPixel = 0
		}
		if rightPixel > src.cols-1 {
			rightPixel = src.cols - 1
		}

		leftPixelCost, midPixelCost, rightPixelCost := src.At(leftPixel, y), src.At(midPixel, y), src.At(rightPixel, y)
		min := math.Min(math.Min(leftPixelCost, midPixelCost), rightPixelCost)

		if min == midPixelCost {
			seam[y] = midPixel
		} else if min == rightPixelCost {
			seam[y] = rightPixel
		} else {
			seam[y] = leftPixel
		}
	}

	return seam
}

func removeSeam(seam []int, costs *matrix[float64], pixels *matrix[color.Color]) {
	for y := 0; y < costs.rows; y++ {
		for x := seam[y]; x < costs.cols-1; x++ {
			costs.Set(x, y, costs.At(x+1, y))
			pixels.Set(x, y, pixels.At(x+1, y))
		}
	}

	costs.cols = costs.cols - 1
	pixels.cols = pixels.cols - 1
}

func drawGrayscaleImage(pixels matrix[float64]) *image.RGBA {
	max := 0.0
	pixels.ForEach(func(x int, y int) {
		max = math.Max(max, pixels.At(x, y))
	})

	rect := image.Rect(0, 0, pixels.cols, pixels.rows)
	dst := image.NewRGBA(rect)

	pixels.ForEach(func(x int, y int) {
		pixel := uint8((pixels.At(x, y) / max) * 255)
		dst.Set(x, y, color.Gray{pixel})
	})

	return dst
}

func drawColorImage(pixels matrix[color.Color]) *image.RGBA {
	rect := image.Rect(0, 0, pixels.cols, pixels.rows)
	dst := image.NewRGBA(rect)

	pixels.ForEach(func(x int, y int) {
		pixel := pixels.At(x, y)
		dst.Set(x, y, pixel)
	})

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
	}

	bounds := img.Bounds()

	// Create slice to store image pixels
	// Operating on this slice is much faster than
	// using image.Image At/Set methods
	imgX, imgY := bounds.Max.X, bounds.Max.Y
	imgMatrix := createMatrix[color.Color](imgY, imgX)
	imgMatrix.ForEach(func(x int, y int) {
		imgMatrix.Set(x, y, img.At(x, y))
	})

	paddedGrayscale := generatePaddedGrayscale(imgMatrix)
	edgeCosts := calculateEdgeCosts(paddedGrayscale)
	costPaths := calculateCostPaths(edgeCosts)

	for i := 0; i < 20; i++ {
		seam := calculateSeam(costPaths)
		removeSeam(seam, &costPaths, &imgMatrix)
	}

	final := drawColorImage(imgMatrix)

	var buff bytes.Buffer
	jpeg.Encode(&buff, final, nil)

	return base64.StdEncoding.EncodeToString(buff.Bytes())
}

func main() {
	js.Global().Set("goCarve", js.FuncOf(carve))
	select {}
}
