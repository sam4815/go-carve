package main

import (
	"bytes"
	"encoding/base64"
	"image"
	"log"
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

	// Make image grayscale to simplify edge detection
	for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
		for x := bounds.Min.X; x < bounds.Max.X; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			grey := uint8(float64(r)*0.2989 + float64(g)*0.5870 + float64(b)*0.1140)
			rgba.Set(x, y, color.RGBA{grey, grey, grey, uint8(a)})
		}
	}
	log.Println("Converted to grayscale")

	var buff bytes.Buffer
	jpeg.Encode(&buff, rgba, nil)

	return base64.StdEncoding.EncodeToString(buff.Bytes())
}

func main() {
	js.Global().Set("goCarve", js.FuncOf(carve))
	select {}
}
