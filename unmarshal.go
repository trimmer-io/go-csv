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

// Package csv decodes and encodes comma-separated values (CSV) files to and from
// arbitrary Go types. Because there are many different kinds of CSV files, this
// package implements the format described in RFC 4180.
//
// A CSV file may contain an optional header and zero or more records of one or
// more fields per record. The number of fields must be the same for each record
// and the optional header. The field separator is configurable and defaults to
// comma ',' (0x2C). Empty lines and lines starting with a comment character
// are ignored. The comment character is configurable as well and defaults to
// the number sign '#' (0x23). Records are separated by the newline character
// '\n' (0x0A) and the final record may or may not be followed by a newline.
// Carriage returns '\r' (0x0D) before newline characters are silently removed.
//
// White space is considered part of a field. Leading or trailing whitespace
// can optionally be trimmed when parsing a value. Fields may optionally be quoted
// in which case the surrounding double quotes '"' (0x22) are removed before
// processing. Inside a quoted field a double quote may be escaped by a preceeding
// second double quote which will be removed during parsing.
//
package csv

import (
	"bufio"
	"bytes"
	"encoding"
	"encoding/hex"
	"fmt"
	"io"
	"reflect"
	"strconv"
	"strings"
)

const (
	Separator = ','
	Comment   = '#'
	Wrapper   = "\""
)

type DecodeError struct {
	lineNo  int
	fieldNo int
	hint    string
	reason  error
}

func (e *DecodeError) Error() string {
	if e.fieldNo != 0 {
		return fmt.Sprintf("csv: line %d field %d (%s): %v", e.lineNo, e.fieldNo, e.hint, e.reason)
	} else if e.reason == nil {
		return fmt.Sprintf("csv: line %d: %s", e.lineNo, e.hint)
	}
	return fmt.Sprintf("csv: line %d: %v", e.lineNo, e.reason)
}

// Unmarshaler is the interface implemented by types that can unmarshal a CSV record
// from a slice of strings. The input is the scanned header array followed by all
// fields for a record. Both slices are guaranteed to be of equal length.
type Unmarshaler interface {
	UnmarshalCSV(header, values []string) error
}

// A Decoder reads and decodes records and fields from a CSV stream.
//
// Using a Decoder is only required when the default behaviour of Unmarshal is undesired.
// This is the case when no headers are present in the CSV file, when special parsing
// is required or for stream processing when files are too large to fit into memory.
//
// When headers are present in a file, a Decoder will interprete the number and order
// of values in each record from the header and map record fields to Go struct fields
// according to their struct tags.
//
// If a header is missing a Decoder will use the type definition of the first value
// passed to DecodeRecord() or the type of slice elements passed to Decode() assuming
// records in the CSV file have the same order as attributes defined for the Go type.
type Decoder struct {
	s           *bufio.Scanner
	sep         rune
	comment     rune
	readHeader  bool
	skipUnknown bool
	trim        bool
	lineNo      int
	headerKeys  []string
}

// NewDecoder returns a new decoder that reads from r.
func NewDecoder(r io.Reader) *Decoder {
	return &Decoder{
		s:           bufio.NewScanner(r),
		readHeader:  true,
		trim:        true,
		skipUnknown: true,
		sep:         Separator,
		comment:     Comment,
		lineNo:      0,
		headerKeys:  make([]string, 0),
	}
}

// Header controls if the decoder expects the input stream to contain header fields.
func (d *Decoder) Header(h bool) *Decoder {
	d.readHeader = h
	return d
}

// Separator sets rune r as record field separator that will be used for parsing.
func (d *Decoder) Separator(r rune) *Decoder {
	d.sep = r
	return d
}

// Comment sets rune c as comment line identifier. Comments must start with rune c
// as first character to be skipped.
func (d *Decoder) Comment(c rune) *Decoder {
	d.comment = c
	return d
}

// Trim controls if the Decoder will trim whitespace surrounding header fields
// and records before processing them.
func (d *Decoder) Trim(t bool) *Decoder {
	d.trim = t
	return d
}

// SkipUnknown controls if the Decoder will return an error when encountering a
// CSV header field that cannot be mapped to a struct tag. When true, such fields will
// be silently ignored in all CSV records.
func (d *Decoder) SkipUnknown(t bool) *Decoder {
	d.skipUnknown = t
	return d
}

// Buffer sets a buffer buf to be used by the underlying bufio.Scanner for reading
// from io.Reader r.
func (d *Decoder) Buffer(buf []byte) *Decoder {
	d.s.Buffer(buf, cap(buf))
	return d
}

// Unmarshal parses CSV encoded data and stores the result in the slice v.
//
// Unmarshal allocates new slice elements for each CSV record encountered
// in the input. The first non-empty and non-commented line of input
// is expected to contain a CSV header that will be used to map the order
// of values in each CSV record to fields in the Go type.
//
// When the slice element type implements the Marshaler interface, UnmarshalCSV
// is called for each record. Otherwise, CSV record fields are assigned to the
// struct fields with a corresponding name in their csv struct tag.
//
//     // CSV field "name" will be assigned to struct field "Field".
//     Field int64 `csv:"name"`
//
//     // Field is used to store all unmapped CSV fields.
//     Field map[string]string `csv:",any"`
//
// A special flag 'any' can be used on a map or any other field type implementing
// TextUnmarshaler interface to capture all unmapped CSV fields of a record.
//
func Unmarshal(data []byte, v interface{}) error {
	return NewDecoder(bytes.NewReader(data)).Decode(v)
}

// ReadLine returns the next non-empty and non-commented line of input. It's
// intended use in combination with DecodeHeader() and DecodeRecord() in loops
// for stream-processing of CSV input. ReadLine returns an error when the
// underlying io.Reader fails. On EOF, ReadLine returns an empty string and
// a nil error.
//
// The canonical way of using ReadLine is (error handling omitted)
//
//      dec := csv.NewDecoder(r)
//      line, _ := dec.ReadLine()
//      head, _ := dec.DecodeHeader(line)
//      for {
//          line, err = dec.ReadLine()
//          if err != nil {
//              return err
//          }
//          if line == "" {
//              break
//          }
//          // process the next record here
//      }
func (d *Decoder) ReadLine() (string, error) {
	for d.s.Scan() {
		line := d.s.Text()
		d.lineNo++
		if len(line) == 0 {
			continue
		}
		if strings.HasPrefix(line, string(d.comment)) {
			continue
		}
		return line, nil
	}
	if err := d.s.Err(); err != nil {
		return "", fmt.Errorf("csv: read failed: %v", err)
	}
	return "", nil
}

// Decode reads CSV records from the input and stores their decoded values in the slice
// pointed to by v.
//
// See the documentation for Unmarshal for details about the conversion of CSV records
// into a Go value.
func (d *Decoder) Decode(v interface{}) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("csv: non-pointer passed to Unmarshal")
	}

	val = reflect.Indirect(val)
	if val.Kind() != reflect.Slice {
		return fmt.Errorf("csv: non-slice passed to Unmarshal")
	}

	// prepare header from type info
	if !d.readHeader {
		tinfo, err := getTypeInfo(indirectType(val.Type().Elem()))
		if err != nil {
			return fmt.Errorf("csv: %v", err)
		}
		for _, finfo := range tinfo.fields {
			if finfo.flags&fAny == 0 {
				d.headerKeys = append(d.headerKeys, finfo.name)
			}
		}
	}

	// everything happens driven by a bufio.Scanner
	for d.s.Scan() {
		line := d.s.Text()
		d.lineNo++

		// skip empty lines
		if len(line) == 0 {
			continue
		}

		// skip comments
		if strings.HasPrefix(line, string(d.comment)) {
			continue
		}

		// process header when not disabled
		if len(d.headerKeys) == 0 && d.readHeader {
			if _, err := d.DecodeHeader(line); err != nil {
				return err
			}
			continue
		}

		// process lines
		e := reflect.New(val.Type().Elem())
		if err := d.unmarshal(e.Elem(), line); err != nil {
			return err
		}

		// append to slice
		val.Set(reflect.Append(val, e.Elem()))
	}
	if err := d.s.Err(); err != nil {
		return fmt.Errorf("csv: read failed: %v", err)
	}

	return nil
}

// DecodeHeader reads CSV head fields from line and stores them as internal
// Decoder state required to map CSV records later on.
func (d *Decoder) DecodeHeader(line string) ([]string, error) {
	d.headerKeys = strings.Split(line, string(d.sep))
	if len(d.headerKeys) == 0 {
		return nil, fmt.Errorf("csv: empty header")
	}
	if d.trim {
		for i, v := range d.headerKeys {
			d.headerKeys[i] = strings.TrimSpace(v)
		}
	}
	return d.headerKeys, nil
}

// DecodeRecord extracts CSV record fields from line and stores them into
// Go value v.
func (d *Decoder) DecodeRecord(v interface{}, line string) error {
	val := reflect.ValueOf(v)
	if val.Kind() != reflect.Ptr {
		return fmt.Errorf("csv: non-pointer passed to DecodeRecord")
	}
	return d.unmarshal(val, line)
}

func (d *Decoder) unmarshal(val reflect.Value, line string) error {
	// split line into tokens
	tokens := strings.Split(line, string(d.sep))

	// combine tokens between ""
	combined := make([]string, 0, len(tokens))
	var merged string
	for _, v := range tokens {
		// unquote and merge multiple tokens, when separated
		switch true {
		case len(v) == 1 && strings.HasPrefix(v, Wrapper):
			// (1) .. ,",", .. (2) .. ," text,", ..
			if merged == "" {
				merged += string(d.sep)
			} else {
				merged += string(d.sep)
				combined = append(combined, merged)
				merged = ""
			}
		case len(v) >= 2 && strings.HasPrefix(v, Wrapper) && strings.HasSuffix(v, Wrapper):
			// (1) .. ,"", .. (2) ..," text text ", ..
			combined = append(combined, v[1:len(v)-1])
			merged = ""
		case strings.HasPrefix(v, Wrapper):
			// .. ," text, more text", .. (1st part)
			merged = v[1:]
		case strings.HasSuffix(v, Wrapper):
			// .. ," text, more text", .. (2nd part)
			merged = strings.Join([]string{merged, v[:len(v)-1]}, string(d.sep))
			combined = append(combined, merged)
			merged = ""
		default:
			// .. ," text, more, text", .. (middle part)
			if merged != "" {
				merged = strings.Join([]string{merged, v}, string(d.sep))
			} else {
				combined = append(combined, v)
			}
		}
	}
	tokens = combined

	if len(tokens) != len(d.headerKeys) {
		return &DecodeError{d.lineNo, 0, "number of fields does not match header", nil}
	}

	// Load value from interface, but only if the result will be
	// usefully addressable.
	val = derefValue(val)

	if val.CanInterface() && val.Type().Implements(unmarshalerType) {
		// This is an unmarshaler with a non-pointer receiver,
		// so it's likely to be incorrect, but we do what we're told.
		return val.Interface().(Unmarshaler).UnmarshalCSV(d.headerKeys, tokens)
	}

	if val.CanAddr() {
		pv := val.Addr()
		if pv.CanInterface() && pv.Type().Implements(unmarshalerType) {
			return pv.Interface().(Unmarshaler).UnmarshalCSV(d.headerKeys, tokens)
		}
	}

	// map struct fields
	for i, fName := range d.headerKeys {
		if d.trim {
			tokens[i] = strings.TrimSpace(tokens[i])
		}

		// remove double quotes
		tokens[i] = strings.Replace(tokens[i], "\"\"", "\"", -1)

		// handle maps
		if val.Kind() == reflect.Map {
			val.SetMapIndex(reflect.ValueOf(fName), reflect.ValueOf(tokens[i]))
			continue
		}

		_, f := d.findStructField(val, fName)
		if !f.IsValid() {
			if d.skipUnknown {
				continue
			} else {
				return &DecodeError{d.lineNo, i + 1, fName, fmt.Errorf("field not found")}
			}
		}

		// try text unmarshalers first
		if f.CanInterface() && f.Type().Implements(textUnmarshalerType) {
			if err := f.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(tokens[i])); err != nil {
				return &DecodeError{d.lineNo, i + 1, fName, err}
			}
			continue
		}

		if f.CanAddr() {
			pv := f.Addr()
			if pv.CanInterface() && pv.Type().Implements(textUnmarshalerType) {
				if err := pv.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(tokens[i])); err != nil {
					return &DecodeError{d.lineNo, i + 1, fName, err}
				}
				continue
			}
		}

		// otherwise set simple value directly
		if err := setValue(f, tokens[i], fName); err != nil {
			return &DecodeError{d.lineNo, i + 1, fName, err}
		}
	}

	return nil
}

func (d *Decoder) findStructField(val reflect.Value, name string) (*fieldInfo, reflect.Value) {
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

func setValue(dst reflect.Value, src, fName string) error {
	if src == "" {
		return nil
	}

	dst0 := dst
	if dst.Kind() == reflect.Ptr {
		if dst.IsNil() {
			dst.Set(reflect.New(dst.Type().Elem()))
		}
		dst = dst.Elem()
	}

	switch dst.Kind() {
	case reflect.Map:
		// map must have map[string]string signature or map value
		// must be an encoding.TextUnmarshaler or a simple value
		t := dst.Type()
		if dst.IsNil() {
			dst.Set(reflect.MakeMap(t))
		}
		switch t.Key().Kind() {
		case reflect.String:
		default:
			return fmt.Errorf("map key type must be string")
		}
		switch t.Elem().Kind() {
		case reflect.String:
			dst.SetMapIndex(reflect.ValueOf(fName), reflect.ValueOf(src).Convert(t.Elem()))
		default:
			// create new map entry and contents if it's pointer type
			val := reflect.New(t.Elem()).Elem()
			if val.Type().Kind() == reflect.Ptr && val.IsNil() && val.CanSet() {
				val.Set(reflect.New(val.Type().Elem()))
			}
			if val.CanInterface() && val.Type().Implements(textUnmarshalerType) {
				if err := val.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(src)); err != nil {
					return err
				}
			} else if val.CanAddr() {
				pv := val.Addr()
				if pv.CanInterface() && pv.Type().Implements(textUnmarshalerType) {
					if err := pv.Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(src)); err != nil {
						return err
					}
				}
			} else {
				if err := setValue(val, src, fName); err != nil {
					return err
				}
			}
			dst.SetMapIndex(reflect.ValueOf(fName), val)
		}

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		i, err := strconv.ParseInt(src, 10, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetInt(i)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		i, err := strconv.ParseUint(src, 10, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetUint(i)
	case reflect.Float32, reflect.Float64:
		i, err := strconv.ParseFloat(src, dst.Type().Bits())
		if err != nil {
			return err
		}
		dst.SetFloat(i)
	case reflect.Bool:
		i, err := strconv.ParseBool(strings.TrimSpace(src))
		if err != nil {
			return err
		}
		dst.SetBool(i)
	case reflect.String:
		dst.SetString(strings.TrimSpace(src))
	case reflect.Slice:
		// make sure it's a byte slice
		if dst.Type().Elem().Kind() == reflect.Uint8 {
			if buf, err := hex.DecodeString(src); err == nil {
				dst.SetBytes(buf)
			} else {
				dst.SetBytes([]byte(src))
			}
		}
	default:
		return fmt.Errorf("no method for unmarshaling type %s", dst0.Type().String())
	}
	return nil
}
