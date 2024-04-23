package binpat_test

import (
	"bytes"
	"encoding/binary"
	"testing"

	"github.com/xypwn/go-binpat"
)

func TestRead(t *testing.T) {
	r := bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 'H', 'e', 'l', 'l', 'o', 0x00, 0xff, 0x00, 0x00, 0x00})
	var data struct {
		Size   uint32
		Data   []byte `binpat:"len=Size"`
		S      string `binpat:"nt"`
		Ignore int32  `binpat:"-"`
		IntLE  int32  `binpat:"le"`
	}
	data.Ignore = 42069
	if err := binpat.Read(r, binary.BigEndian, &data); err != nil {
		t.Fatal(err)
	}
	if data.Size != 8 ||
		!bytes.Equal(data.Data, []byte{0, 1, 2, 3, 4, 5, 6, 7}) ||
		data.S != "Hello" ||
		data.Ignore != 42069 ||
		data.IntLE != 0xff {
		t.Fatalf("unexpected value: %+v\n", data)
	}
}

func TestPassthroughRead(t *testing.T) {
	r := bytes.NewReader([]byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04})
	var data [4]int32
	if err := binpat.Read(r, binary.BigEndian, &data); err != nil {
		t.Fatal(err)
	}
	if data != [4]int32{1, 2, 3, 4} {
		t.Fatalf("unexpected value: %+v", data)
	}
}

func TestWrite(t *testing.T) {
	var w bytes.Buffer
	data := struct {
		Size   uint32
		Data   []byte `binpat:"len=Size"`
		S      string `binpat:"nt"`
		Ignore int32  `binpat:"-"`
		IntLE  int32  `binpat:"le"`
	}{
		Size:   8,
		Data:   []byte{0, 1, 2, 3, 4, 5, 6, 7},
		S:      "Hello",
		Ignore: 42069,
		IntLE:  0xff,
	}
	if err := binpat.Write(&w, binary.BigEndian, data); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(w.Bytes(), []byte{0x00, 0x00, 0x00, 0x08, 0x00, 0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 'H', 'e', 'l', 'l', 'o', 0x00, 0xff, 0x00, 0x00, 0x00}) {
		t.Fatalf("unexpected value: %+v\n", w.Bytes())
	}
}

func TestPassthroughWrite(t *testing.T) {
	var w bytes.Buffer
	data := [4]int32{1, 2, 3, 4}
	if err := binpat.Write(&w, binary.BigEndian, data); err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(w.Bytes(), []byte{0x00, 0x00, 0x00, 0x01, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00, 0x00, 0x03, 0x00, 0x00, 0x00, 0x04}) {
		t.Fatalf("unexpected value: %+v", w.Bytes())
	}
}
