package main

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"unicode"

	"github.com/xypwn/go-binpat"
)

func main() {
	var w bytes.Buffer
	data := struct {
		Size uint32
		Data []byte `binpat:"len=Size"`
		Str  string `binpat:"nt"`
	}{
		Size: 8, // must equal len(Data) or an error occurs
		Data: []byte{0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07},
		Str:  "Hello",
	}
	if err := binpat.Write(&w, binary.BigEndian, data); err != nil {
		panic(err)
	}
	fmt.Printf("Data: %+v\n", data)

	b := w.Bytes()
	fmt.Print("Bytes: [")
	for i := range b {
		if i != 0 {
			fmt.Print(" ")
		}
		fmt.Printf("0x%02x", b[i])
		if unicode.IsPrint(rune(b[i])) {
			fmt.Printf("('%c')", b[i])
		}
	}
	fmt.Println("]")
}
