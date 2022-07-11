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

func (matrix *matrix[T]) Transpose() {
	vals := make([]T, matrix.rows*matrix.cols)

	// Transpose the matrix
	for y := 0; y < matrix.rows; y++ {
		for x := 0; x < matrix.cols; x++ {
			vals[y+x*matrix.rows] = matrix.vals[x+y*matrix.initCols]
		}
	}

	matrix.vals = vals
	matrix.rows, matrix.cols, matrix.initCols = matrix.cols, matrix.rows, matrix.rows
}

func (matrix *matrix[T]) FlipVertical() {
	for y := 0; y < matrix.rows/2; y++ {
		upperRowIndex := y * matrix.initCols
		lowerRowIndex := (matrix.rows - y - 1) * matrix.initCols

		for x := 0; x < matrix.cols; x++ {
			matrix.vals[x+upperRowIndex], matrix.vals[x+lowerRowIndex] = matrix.vals[x+lowerRowIndex], matrix.vals[x+upperRowIndex]
		}
	}
}

func (matrix *matrix[T]) RotateRight() {
	matrix.FlipVertical()
	matrix.Transpose()
}

func (matrix *matrix[T]) RotateLeft() {
	matrix.Transpose()
	matrix.FlipVertical()
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

func calculateCostPaths(costs *matrix[float64], paths *matrix[float64]) {
	paths.rows, paths.cols, paths.initCols = costs.rows, costs.cols, costs.initCols

	for y := costs.rows - 1; y >= 0; y-- {
		for x := 0; x < costs.cols; x++ {
			currentPixelCost := costs.At(x, y)
			// Handle final row
			if y == costs.rows-1 {
				paths.Set(x, y, currentPixelCost)
				continue
			}

			leftPixelIndex, midPixelIndex, rightPixelIndex := x-1, x, x+1
			if x == 0 {
				leftPixelIndex = 0
			} else if x == costs.cols-1 {
				rightPixelIndex = costs.cols - 1
			}

			minCost := math.Min(math.Min(paths.At(leftPixelIndex, y+1), paths.At(midPixelIndex, y+1)), paths.At(rightPixelIndex, y+1))
			cost := minCost + currentPixelCost

			paths.Set(x, y, cost)
		}
	}
}

func calculateSeam(src *matrix[float64]) []int {
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

func removeSeam(seam []int, costs *matrix[float64], colorImg *matrix[color.Color]) {
	for y := 0; y < colorImg.rows; y++ {
		for x := seam[y]; x < colorImg.cols-1; x++ {
			colorImg.Set(x, y, colorImg.At(x+1, y))
			costs.Set(x, y, costs.At(x+1, y))
		}
	}

	colorImg.cols = colorImg.cols - 1
	costs.cols = costs.cols - 1
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

func convertImageToMatrix(image image.Image) matrix[color.Color] {
	bounds := image.Bounds()
	imgX, imgY := bounds.Max.X, bounds.Max.Y

	imageMatrix := createMatrix[color.Color](imgY, imgX)

	imageMatrix.ForEach(func(x int, y int) {
		imageMatrix.Set(x, y, image.At(x, y))
	})

	return imageMatrix
}

func readBase64Image(src string) image.Image {
	srcBytes, err := base64.StdEncoding.DecodeString(src)
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

	return img
}

func encodeAsJPEGString(src image.Image) string {
	var buffer bytes.Buffer
	jpeg.Encode(&buffer, src, nil)

	return base64.StdEncoding.EncodeToString(buffer.Bytes())
}

func detectEdges(this js.Value, inputs []js.Value) interface{} {
	imageMatrix := convertImageToMatrix(readBase64Image(inputs[0].String()))
	paddedGrayscale := generatePaddedGrayscale(imageMatrix)
	edgeCosts := calculateEdgeCosts(paddedGrayscale)

	return encodeAsJPEGString(drawGrayscaleImage(edgeCosts))
}

func calculatePaths(this js.Value, inputs []js.Value) interface{} {
	imageMatrix := convertImageToMatrix(readBase64Image(inputs[0].String()))
	paddedGrayscale := generatePaddedGrayscale(imageMatrix)
	edgeCosts := calculateEdgeCosts(paddedGrayscale)
	costPaths := createMatrix[float64](edgeCosts.rows, edgeCosts.cols)
	calculateCostPaths(&edgeCosts, &costPaths)

	return encodeAsJPEGString(drawGrayscaleImage(costPaths))
}

func removeSeams(numSeams int, imageMatrix *matrix[color.Color], edgeCosts *matrix[float64], costPaths *matrix[float64]) {
	for i := 0; i < numSeams; i++ {
		calculateCostPaths(edgeCosts, costPaths)
		seam := calculateSeam(costPaths)
		removeSeam(seam, edgeCosts, imageMatrix)

		if (i+1)%50 == 0 {
			log.Println(i+1, " seams removed")
		}
	}
}

func carve(this js.Value, inputs []js.Value) interface{} {
	image := readBase64Image(inputs[0].String())
	imageMatrix := convertImageToMatrix(image)

	dstRows, dstCols := inputs[1].Int(), inputs[2].Int()
	numXSeams := imageMatrix.cols - dstCols
	numYSeams := imageMatrix.rows - dstRows

	log.Println("Removing ", numXSeams, " vertical seams and ", numYSeams, " horizontal seams.")

	paddedGrayscale := generatePaddedGrayscale(imageMatrix)
	edgeCosts := calculateEdgeCosts(paddedGrayscale)
	costPaths := createMatrix[float64](edgeCosts.rows, edgeCosts.cols)

	removeSeams(numXSeams, &imageMatrix, &edgeCosts, &costPaths)

	if numYSeams > 0 {
		edgeCosts.RotateRight()
		imageMatrix.RotateRight()

		removeSeams(numYSeams, &imageMatrix, &edgeCosts, &costPaths)

		imageMatrix.RotateLeft()
	}

	return encodeAsJPEGString(drawColorImage(imageMatrix))
}

func main() {
	js.Global().Set("goCarve", js.FuncOf(carve))
	js.Global().Set("goDetectEdges", js.FuncOf(detectEdges))
	js.Global().Set("goCalculatePaths", js.FuncOf(calculatePaths))
	select {}
}
