package main

import "syscall/js"

func main() {
	doc := js.Global().Get("document")
	body := doc.Get("body")

	h1 := doc.Call("createElement", "h1")
	h1.Set("innerHTML", "Working!")

	body.Call("append", h1)
}
