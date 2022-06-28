package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"log"
	"syscall/js"

	"image/draw"
	"image/jpeg"
	_ "image/png"
)

func setImage(img image.Image) {
	doc := js.Global().Get("document")
	imageEl := doc.Call("getElementById", "output")

	var buff bytes.Buffer
	jpeg.Encode(&buff, img, nil)

	imageEl.Set("src", fmt.Sprintf("data:image/jpeg;base64,%s", base64.StdEncoding.EncodeToString(buff.Bytes())))
}

func analyze(this js.Value, inputs []js.Value) interface{} {
	imageArr := inputs[0]
	srcBytes := make([]uint8, imageArr.Get("byteLength").Int())
	js.CopyBytesToGo(srcBytes, imageArr)

	reader := bytes.NewReader(srcBytes)
	img, _, err := image.Decode(reader)
	if err != nil {
		log.Println(err)
		return nil
	}
	// bounds := img.Bounds()

	rgba, ok := img.(*image.RGBA)
	if !ok {
		b := img.Bounds()
		rgba = image.NewRGBA(image.Rect(0, 0, b.Dx(), b.Dy()))
		draw.Draw(rgba, rgba.Bounds(), img, b.Min, draw.Src)
	}

	// for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
	// 	for x := bounds.Min.X; x < bounds.Max.X; x++ {
	// 		r, g, b, a := img.At(x, y).RGBA()
	// 		avg := uint8((r + g + b) / 3)
	// 		rgba.Set(x, y, color.RGBA{avg, avg, avg, uint8(a)})
	// 	}
	// }

	setImage(rgba)

	return imageArr
}

func main() {
	js.Global().Set("analyze", js.FuncOf(analyze))
	select {}
}
