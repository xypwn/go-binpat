<div align="center">

# Go-Binpat

[![Go Report Card](https://goreportcard.com/badge/github.com/xypwn/go-binpat)](https://goreportcard.com/report/github.com/xypwn/go-binpat)
[![GitHub License](https://img.shields.io/github/license/xypwn/go-binpat)](https://opensource.org/license/mit)

Drop-in replacement for the standard binary package that allows for defining more complex structures using struct tags.
</div>

### Example: Read
```go
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
```

Output:
```
{Size:8 Data:[0 1 2 3 4 5 6 7] Str:Hello}
```

### Example: Write
```go
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
```

Output:
```
Data: {Size:8 Data:[0 1 2 3 4 5 6 7] Str:Hello}
Bytes: [0x00 0x00 0x00 0x08 0x00 0x01 0x02 0x03 0x04 0x05 0x06 0x07 0x48('H') 0x65('e') 0x6c('l') 0x6c('l') 0x6f('o') 0x00]
```

### Struct tags
#### Format by example
```go
struct {
    Size uint32
    Data []int32 `binpat:"be,len=Size"`
    Str  string  `binpat:"nt"`
}
```

#### List of struct tags
| Tag                           | Field type          | Inherited | Description                         |
|-------------------------------|---------------------|-----------|-------------------------------------|
| `-`                           | `any`               | ❌        | do not serialize field              |
| `le`                          | `any`               | ✔️        | always interpret as little endian   |
| `be`                          | `any`               | ✔️        | always interpret as big endian      |
| `ne`                          | `any`               | ✔️        | always interpret as native endian   |
| `nt`                          | `string`            | ❌        | string is null-terminated           |
| `len=<FieldName>`             | `[]any` \| `string` | ❌        | slice/string length                 |