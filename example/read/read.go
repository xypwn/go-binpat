package main

import (
	"bytes"
	"encoding/binary"
	"fmt"

	"github.com/xypwn/go-binpat"
)

func main() {
	r := bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 'H', 'e', 'l', 'l', 'o', 0x00})
	var data struct {
		Size uint32
		Data []byte `binpat:"len=Size"`
		Str  string `binpat:"nt"`
	}
	if err := binpat.Read(r, binary.BigEndian, &data); err != nil {
		panic(err)
	}
	fmt.Printf("%+v\n", data)
}
