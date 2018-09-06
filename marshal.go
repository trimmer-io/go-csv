// Copyright (c) 2017 Alexander Eichhorn
//
// Licensed under the Apache License, Version 2.0 (the "License"): you may
// not use this file except in compliance with the License. You may obtain
// a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
// WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
// License for the specific language governing permissions and limitations
// under the License.

package csv // import "trimmer.io/go-csv"

import (
	"bytes"
	"encoding"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

// Marshaler is the interface implemented by types that can marshal themselves
// as CSV records. The assumed return value is a slice of strings that must be
// of same length for all records and the header.
type Marshaler interface {
	MarshalCSV() ([]string, error)
}

// Encoder writes CSV header and CSV records to an output stream. The encoder
// may be configured to omit the header, to use a user-defined separator and
// to trim string values before writing them as CSV fields.
type Encoder struct {
	w           io.Writer
	sep         string
	trim        bool
	writeHeader bool
	headerKeys  []string
}

// NewEncoder returns a new encoder that writes to w.
func NewEncoder(w io.Writer) *Encoder {
	return &Encoder{
		w:           w,
		sep:         string(Separator),
		trim:        true,
		writeHeader: true,
	}
}

// Header controls if the encoder will write a CSV header to the first line of
// the output stream.
func (e *Encoder) Header(h bool) *Encoder {
	e.writeHeader = h
	return e
}

// Separator sets the rune r that will be used to separate header fields and
// CSV record fields.
func (e *Encoder) Separator(r rune) *Encoder {
	e.sep = string(r)
	return e
}

// Trim controls if the Decoder will trim whitespace surrounding string values
// before writing them to the output stream.
func (e *Encoder) Trim(t bool) *Encoder {
	e.trim = t
	return e
}

// Marshal returns the CSV encoding of slice v.
//
// When the slice's element type implements the Marshaler interface, MarshalCSV
// is called for each element and the resulting string slice is written in the
// order returned by MarshalCSV to the output stream. Otherwise, CSV records are
// ordered like type attributes in the element's type definition.
//
// CSV header field names are taken from struct field tags of each attribute and
// when missing from the attribute name as specified in the Go type.
//
//     // CSV field "name" will be assigned to struct field "Field".
//     Field int64 `csv:"name"`
//
//     // Field is ignored by this package.
//     Field int `csv:"-"`
//
// Marshal only supports strings, integers, floats, booleans, []byte slices
// and [N]byte arrays as well as pointers to these types. Slices of other
// types, maps, interfaces and channels are not supported and result in an
// error when passed to Marshal.
func Marshal(v interface{}) ([]byte, error) {
	var b bytes.Buffer
	if err := NewEncoder(&b).Encode(v); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// Encode writes the CSV encoding of slice v to the stream.
//
// See the documentation for Marshal for details about the conversion of Go values
// to CSV.
func (e *Encoder) Encode(v interface{}) error {
	val := reflect.Indirect(reflect.ValueOf(v))
	if !val.IsValid() {
		return nil
	}

	if val.Kind() != reflect.Slice {
		return fmt.Errorf("csv: non-slice type passed to Marshal: %s %s", val.Kind().String(), val.Type().String())
	}

	// always prepare header, but write only when requested
	if err := e.EncodeHeader(nil, val.Index(0).Interface()); err != nil {
		return err
	}

	// process records
	for i, l := 0, val.Len(); i < l; i++ {
		if err := e.EncodeRecord(val.Index(i).Interface()); err != nil {
			return err
		}
	}
	return nil
}

// EncodeHeader prepares and optionally writes a CSV header. When fields is not
// empty, it determines which header fields and subsequently which attributes
// from a Go type will be written as CSV record fields.
//
// When fields is nil or empty, the value of v will be used to determine the
// type of records and their field names. v in this case is an element of the
// slice you would pass to Marshal, not a slice itself.
func (e *Encoder) EncodeHeader(fields []string, v interface{}) error {
	if err := e.buildHeader(fields, reflect.ValueOf(v)); err != nil {
		return err
	}
	if !e.writeHeader {
		return nil
	}
	return e.output(e.headerKeys)
}

// EncodeRecord writes the CSV encoding of v to the output stream.
func (e *Encoder) EncodeRecord(v interface{}) error {
	if len(e.headerKeys) == 0 {
		if err := e.EncodeHeader(nil, v); err != nil {
			return err
		}
	}
	if err := e.marshal(reflect.ValueOf(v)); err != nil {
		return fmt.Errorf("csv: %v", err)
	}
	return nil
}

func (e *Encoder) buildHeader(fields []string, val reflect.Value) error {
	// build only once
	if len(e.headerKeys) > 0 {
		return nil
	}
	// use user-provided fields if set
	if len(fields) > 0 {
		e.headerKeys = fields
		return nil
	}
	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}
	tinfo, err := getTypeInfo(val.Type())
	if err != nil {
		return fmt.Errorf("csv: %v", err)
	}
	e.headerKeys = make([]string, len(tinfo.fields))
	for i, finfo := range tinfo.fields {
		e.headerKeys[i] = finfo.name
	}
	return nil
}

func (e *Encoder) output(fields []string) error {
	line := strings.Join(fields, string(e.sep))
	if _, err := e.w.Write([]byte(line)); err != nil {
		return fmt.Errorf("csv: %v", err)
	}
	if _, err := e.w.Write([]byte("\n")); err != nil {
		return fmt.Errorf("csv: %v", err)
	}
	return nil

}

func (e *Encoder) marshal(val reflect.Value) error {
	// Load value from interface
	val = derefValue(val)

	if val.CanInterface() && val.Type().Implements(marshalerType) {
		fields, err := val.Interface().(Marshaler).MarshalCSV()
		if err != nil {
			return err
		}
		return e.output(fields)
	}

	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(marshalerType) {
			fields, err := pv.Interface().(Marshaler).MarshalCSV()
			if err != nil {
				return err
			}
			return e.output(fields)
		}
	}

	if val.Kind() == reflect.Ptr && !val.IsNil() {
		val = val.Elem()
	}

	// map struct fields
	tokens := make([]string, len(e.headerKeys))

	// work with []string, []interface{} and other slices with types than
	// convert to string
	if val.Kind() == reflect.Slice && val.Len() == len(e.headerKeys) {
		for i := 0; i < val.Len(); i++ {
			f := val.Index(i)
			if f.IsNil() {
				continue
			}
			if f.Type().Kind() == reflect.Interface {
				f = f.Elem()
			}
			if f.Type().Kind() == reflect.Ptr && !f.IsNil() {
				f = f.Elem()
			}
			if !f.IsValid() {
				continue
			}
			s, b, err := marshalSimple(f.Type(), f)
			if err != nil {
				return err
			}
			if b != nil {
				s = string(b)
			}
			tokens[i] = s
		}
	} else {
		for i, fName := range e.headerKeys {
			// init with empty string
			tokens[i] = ""

			finfo, f := e.findStructField(val, fName)
			if finfo == nil || !f.IsValid() {
				continue
			}

			if finfo.flags&fElement == 0 {
				continue
			}

			fv := finfo.value(val)

			if (fv.Kind() == reflect.Interface || fv.Kind() == reflect.Ptr) && fv.IsNil() {
				continue
			}

			// try text marshalers first
			if fv.CanInterface() && fv.Type().Implements(textMarshalerType) {
				if b, err := fv.Interface().(encoding.TextMarshaler).MarshalText(); err != nil {
					return err
				} else {
					tokens[i] = string(b)
				}
				continue
			}

			if f.CanAddr() {
				pv := f.Addr()
				if pv.CanInterface() && pv.Type().Implements(textMarshalerType) {
					if b, err := pv.Interface().(encoding.TextMarshaler).MarshalText(); err != nil {
						return err
					} else {
						tokens[i] = string(b)
					}
				}
			}
			s, b, err := marshalSimple(f.Type(), f)
			if err != nil {
				return err
			}
			if b != nil {
				s = string(b)
			}
			tokens[i] = s

			// trim
			if e.trim {
				tokens[i] = strings.TrimSpace(tokens[i])
			}
		}
	}
	return e.output(tokens)
}

func (e *Encoder) findStructField(val reflect.Value, name string) (*fieldInfo, reflect.Value) {
	typ := val.Type()
	tinfo, err := getTypeInfo(typ)
	if err != nil {
		return nil, reflect.Value{}
	}

	var finfo *fieldInfo
	any := -1
	// pick the correct field based on name and flags
	for i, v := range tinfo.fields {
		// save `any` field in case
		if v.flags&fAny > 0 {
			any = i
		}

		// field name must match
		if v.name != name {
			continue
		}

		finfo = &v
		break
	}

	if finfo == nil && any >= 0 {
		finfo = &tinfo.fields[any]
	}

	// nothing found
	if finfo == nil {
		return nil, reflect.Value{}
	}

	// allocate memory for pointer values in structs
	v := finfo.value(val)
	if v.Type().Kind() == reflect.Ptr && v.IsNil() && v.CanSet() {
		v.Set(reflect.New(v.Type().Elem()))
	}

	return finfo, v
}

var stringerType = reflect.TypeOf((*fmt.Stringer)(nil)).Elem()

func marshalSimple(typ reflect.Type, val reflect.Value) (string, []byte, error) {
	if typ.Implements(stringerType) {
		return val.Interface().(fmt.Stringer).String(), nil, nil
	}
	switch val.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return strconv.FormatUint(val.Uint(), 10), nil, nil
	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'g', -1, val.Type().Bits()), nil, nil
	case reflect.String:
		return val.String(), nil, nil
	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil, nil
	case reflect.Array:
		if typ.Elem().Kind() != reflect.Uint8 {
			break
		}
		// [...]byte
		var bytes []byte
		if val.CanAddr() {
			bytes = val.Slice(0, val.Len()).Bytes()
		} else {
			bytes = make([]byte, val.Len())
			reflect.Copy(reflect.ValueOf(bytes), val)
		}
		return "", bytes, nil
	case reflect.Slice:
		if typ.Elem().Kind() != reflect.Uint8 {
			break
		}
		// []byte
		return "", val.Bytes(), nil
	}
	return "", nil, fmt.Errorf("no method for marshalling type %s (%v)", typ.String(), val.Kind())
}
