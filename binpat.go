// Package binpat is a drop-in replacement for the standard binary package that allows for defining more complex structures using struct tags.
package binpat

import (
	"encoding/binary"
	"errors"
	"io"
	"math"
	"reflect"
	"strings"
)

type context struct {
	TagStr   string           // full tag string for error messages
	Order    binary.ByteOrder // inherited
	SkipThis bool
	LenField string
	NullTerm bool
}

// Inherits "inherited" fields from parent.
func contextFromTag(parent context, tag string) context {
	ctx := context{
		TagStr: tag,
		Order:  parent.Order,
	}
	if tag == "" {
		return ctx
	}
	items := strings.Split(tag, ",")
	for _, item := range items {
		key, val, hasVal := strings.Cut(item, "=")
		shouldHaveVal := false
		switch key {
		case "le":
			ctx.Order = binary.LittleEndian
		case "be":
			ctx.Order = binary.BigEndian
		case "ne":
			ctx.Order = binary.NativeEndian
		case "nt":
			ctx.NullTerm = true
		case "len":
			shouldHaveVal = true
			ctx.LenField = val
		case "-":
			ctx.SkipThis = true
		default:
			panic("binpat: \"" + tag + "\": unrecognized key " + key)
		}
		if shouldHaveVal && !hasVal {
			panic("binpat: \"" + tag + "\": " + key + " needs a value")
		}
		if !shouldHaveVal && hasVal {
			panic("binpat: \"" + tag + "\": " + key + " cannot have a value")
		}
	}
	return ctx
}

func (c context) fieldMustValid(kind reflect.Kind, fieldName string) {
	if c.SkipThis {
		if c.TagStr != "-" {
			panic("binpat: \"" + c.TagStr + "\": if \"-\" is used, no other tags may be used")
		}
	}
	if c.NullTerm && c.LenField != "" {
		panic("binpat: \"" + c.TagStr + "\": can only have one of len or nt")
	}
	if c.LenField != "" {
		if kind != reflect.Slice && kind != reflect.String {
			panic("binpat: \"" + c.TagStr + "\": len tag must be applied to a slice or string")
		}
	}
	if c.NullTerm {
		if kind != reflect.String {
			panic("binpat: \"" + c.TagStr + "\": nt tag must be applied to a string")
		}
	}
	if kind == reflect.String {
		if !c.NullTerm && c.LenField == "" {
			panic("binpat: string field " + fieldName + " must have a len or nt property")
		}
	}
	if kind == reflect.Slice {
		if c.LenField == "" {
			panic("binpat.Read: slice field " + fieldName + " must have a len property")
		}
	}
}

func getStructFieldInfo(ctx context, v reflect.Value) (childCtxs []context, fieldIdxs map[string]int, unexportedFields map[string]struct{}, isLenField map[int]struct{}) {
	vType := v.Type()
	childCtxs = make([]context, v.NumField())
	fieldIdxs = make(map[string]int)
	unexportedFields = make(map[string]struct{})
	for i := 0; i < v.NumField(); i++ {
		fInfo := vType.Field(i)
		if !fInfo.IsExported() {
			unexportedFields[fInfo.Name] = struct{}{}
			continue
		}
		tag, _ := fInfo.Tag.Lookup("binpat")
		ctx := contextFromTag(ctx, tag)
		childCtxs[i] = ctx
		fieldIdxs[fInfo.Name] = i
	}
	isLenField = make(map[int]struct{})
	for i := 0; i < v.NumField(); i++ {
		isLenField[fieldIdxs[ctx.LenField]] = struct{}{}
	}
	return
}

func lenFieldToInt(v reflect.Value) (int, error) {
	val := 0
	if v.CanInt() {
		x := v.Int()
		if x > math.MaxInt || x < math.MinInt {
			return 0, errors.New("binpat.Read: slice length too large / small")
		}
		if x >= 0 {
			val = int(x)
		}
	} else if v.CanUint() {
		x := v.Uint()
		if x > math.MaxUint {
			return 0, errors.New("binpat.Read: slice length too large")
		}
		val = int(x)
	} else {
		return 0, errors.New("binpat.Read: invalid length type " + v.Type().String())
	}
	return val, nil
}

func getSliceFieldLen(ctx context, sliceFieldIdx int, fieldIdxs map[string]int, unexportedFields map[string]struct{}, lenFieldVals map[int]int) int {
	if ctx.LenField == "" {
		panic("getSliceFieldLen called on slice field without len")
	}
	if _, ok := unexportedFields[ctx.LenField]; ok {
		panic("binpat: \"" + ctx.TagStr + "\": referenced field " + ctx.LenField + " is not exported")
	}
	lenFieldIdx, ok := fieldIdxs[ctx.LenField]
	if !ok {
		panic("binpat: \"" + ctx.TagStr + "\": field " + ctx.LenField + " does not exist")
	}
	if lenFieldIdx >= sliceFieldIdx {
		panic("binpat: \"" + ctx.TagStr + "\": len field " + ctx.LenField + " must come before value field")
	}
	return lenFieldVals[lenFieldIdx]
}

type Binpat struct {
	Funcs map[string]func(any) any
}

func (bp *Binpat) read(ctx context, r io.Reader, data any) error {
	switch data.(type) {
	// Pass fast non-struct cases directly to binary.Read
	case bool, int8, uint8, *bool, *int8, *uint8, []bool, []int8, []uint8, int16, uint16, *int16, *uint16, []int16, []uint16, int32, uint32, *int32, *uint32, []int32, []uint32, int64, uint64, *int64, *uint64, []int64, []uint64, float32, *float32, float64, *float64, []float32, []float64:
		return binary.Read(r, ctx.Order, data)
	}
	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	vType := v.Type()
	if v.Kind() == reflect.Struct {
		childCtxs, fieldIdxs, unexportedFields, isLenField := getStructFieldInfo(ctx, v)

		lenFieldVals := make(map[int]int)

		for i := 0; i < v.NumField(); i++ {
			fInfo := vType.Field(i)
			if !fInfo.IsExported() {
				continue
			}
			fV := v.Field(i)
			if fV.Kind() == reflect.Pointer {
				fV = fV.Elem()
			}
			ctx := childCtxs[i]
			ctx.fieldMustValid(fInfo.Type.Kind(), fInfo.Name)
			if ctx.SkipThis {
				continue
			}

			switch fV.Kind() {
			case reflect.Slice:
				sliceLen := getSliceFieldLen(ctx, i, fieldIdxs, unexportedFields, lenFieldVals)
				fV.Set(reflect.MakeSlice(fV.Type(), sliceLen, sliceLen))
				for i := 0; i < sliceLen; i++ {
					if err := bp.read(ctx, r, fV.Index(i).Addr().Interface()); err != nil {
						return err
					}
				}
			case reflect.String:
				if ctx.NullTerm {
					var s strings.Builder
					for {
						var buf [1]byte
						if _, err := io.ReadFull(r, buf[:]); err != nil {
							return err
						}
						if buf[0] == 0 {
							break
						}
						if err := s.WriteByte(buf[0]); err != nil {
							return err
						}
					}
					fV.SetString(s.String())
				} else if ctx.LenField != "" {
					sliceLen := getSliceFieldLen(ctx, i, fieldIdxs, unexportedFields, lenFieldVals)
					buf := make([]byte, sliceLen)
					if _, err := io.ReadFull(r, buf); err != nil {
						return err
					}
					fV.SetString(string(buf))
				}
			default:
				if err := bp.read(ctx, r, fV.Addr().Interface()); err != nil {
					return err
				}
			}
			if _, ok := isLenField[i]; ok {
				val, err := lenFieldToInt(fV)
				if err != nil {
					return err
				}
				lenFieldVals[i] = val
			}
		}
	} else {
		return binary.Read(r, ctx.Order, data)
	}
	return nil
}

func (bp *Binpat) Read(r io.Reader, order binary.ByteOrder, data any) error {
	return bp.read(context{Order: order}, r, data)
}

func Read(r io.Reader, order binary.ByteOrder, data any) error {
	return (&Binpat{}).Read(r, order, data)
}

func (bp *Binpat) write(ctx context, w io.Writer, data any) error {
	switch data.(type) {
	// Pass fast non-struct cases directly to binary.Write
	case bool, int8, uint8, *bool, *int8, *uint8, []bool, []int8, []uint8, int16, uint16, *int16, *uint16, []int16, []uint16, int32, uint32, *int32, *uint32, []int32, []uint32, int64, uint64, *int64, *uint64, []int64, []uint64, float32, *float32, float64, *float64, []float32, []float64:
		return binary.Write(w, ctx.Order, data)
	}

	v := reflect.ValueOf(data)
	if v.Kind() == reflect.Pointer {
		v = v.Elem()
	}
	vType := v.Type()
	if v.Kind() == reflect.Struct {
		childCtxs, fieldIdxs, unexportedFields, isLenField := getStructFieldInfo(ctx, v)

		lenFieldVals := make(map[int]int)
		for i := 0; i < v.NumField(); i++ {
			fV := v.Field(i)
			if _, ok := isLenField[i]; ok {
				val, err := lenFieldToInt(fV)
				if err != nil {
					return err
				}
				lenFieldVals[i] = val
			}
		}

		for i := 0; i < v.NumField(); i++ {
			fInfo := vType.Field(i)
			if !fInfo.IsExported() {
				continue
			}
			fV := v.Field(i)
			if fV.Kind() == reflect.Pointer {
				fV = fV.Elem()
			}
			ctx := childCtxs[i]
			ctx.fieldMustValid(fInfo.Type.Kind(), fInfo.Name)
			if ctx.SkipThis {
				continue
			}

			switch fV.Kind() {
			case reflect.Slice:
				if ctx.LenField != "" && getSliceFieldLen(ctx, i, fieldIdxs, unexportedFields, lenFieldVals) != fV.Len() {
					return errors.New("binpat: slice " + fInfo.Name + " must have len of " + ctx.LenField)
				}
				for i := 0; i < fV.Len(); i++ {
					if err := bp.write(ctx, w, fV.Index(i).Interface()); err != nil {
						return err
					}
				}
			case reflect.String:
				if ctx.LenField != "" && getSliceFieldLen(ctx, i, fieldIdxs, unexportedFields, lenFieldVals) != fV.Len() {
					return errors.New("binpat: string " + fInfo.Name + " must have len of " + ctx.LenField)
				}
				if _, err := w.Write([]byte(fV.String())); err != nil {
					return err
				}
				if ctx.NullTerm {
					if _, err := w.Write([]byte{0}); err != nil {
						return err
					}
				}
			default:
				if err := bp.write(ctx, w, fV.Interface()); err != nil {
					return err
				}
			}
		}
	} else {
		return binary.Write(w, ctx.Order, data)
	}
	return nil
}

func (bp *Binpat) Write(w io.Writer, order binary.ByteOrder, data any) error {
	return bp.write(context{Order: order}, w, data)
}

func Write(w io.Writer, order binary.ByteOrder, data any) error {
	return (&Binpat{}).Write(w, order, data)
}
