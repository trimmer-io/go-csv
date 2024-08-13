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

package csv

import (
	"bytes"
	"io"
	"strconv"
	"testing"
)

type A struct {
	String string  `csv:"s"`
	Bool   bool    `csv:"b"`
	Int    int64   `csv:"i"`
	Float  float64 `csv:"f"`
}

type B struct {
	String string            `csv:"s"`
	Bool   bool              `csv:"b"`
	Int    int64             `csv:"i"`
	Float  float64           `csv:"f"`
	Any    map[string]string `csv:",any"`
}

type C struct {
	String string             `csv:"s"`
	Bool   bool               `csv:"b"`
	Int    int64              `csv:"i"`
	Float  float64            `csv:"f"`
	Any    map[string]Special `csv:",any"`
}

type D struct {
	String string                    `csv:"s"`
	Bool   bool                      `csv:"b"`
	Int    int64                     `csv:"i"`
	Float  float64                   `csv:"f"`
	Any    map[string]*SpecialStruct `csv:",any"`
}

type Special string

func (x *Special) UnmarshalText(b []byte) error {
	*x = Special(b)
	return nil
}

func (x Special) String() string {
	return string(x)
}

type SpecialStruct struct {
	s string
}

func (x *SpecialStruct) UnmarshalText(b []byte) error {
	x.s = string(b)
	return nil
}

func (x SpecialStruct) String() string {
	return x.s
}

const (
	CsvWithHeader = `s,i,f,b
Hello,42,23.45,true`
	CsvWithoutHeader = `Hello,true,42,23.45`
	CsvQuoted        = `"Hello",true,42,23.45`
	CsvWhitespace    = `  Hello  ,  true   ,  42  ,  23.45`
	CsvSemicolon     = `Hello;true;42;23.45`
	CsvComment       = `# Comment line
Hello,true,42,23.45
#
# another comment
Hello World,false,43,24.56`
	CsvEmptyLine = `
Hello,true,42,23.45

Hello World,false,43,24.56`
	CsvEmptyField = `,true,42,23.45
Hello,,42,23.45
Hello,true,,23.45
Hello,true,42,
,,,`
	CsvUnknownField = `s,i,f,b,x
Hello,42,23.45,true,Unknown`
	CsvAnyFields = `s,i,f,b,x,y
Hello,42,23.45,true,X,Y`
	CsvWithCRLF  = "s,i,f,b\r\nHello,42,23.45,true\r\nHello World,43,24.56,false\r\n"
	CsvWithoutLF = "s,i,f,b\nHello,42,23.45,true\nHello World,43,24.56,false"
)

var (
	A1 = A{"Hello", true, 42, 23.45}
	A2 = A{"Hello World", false, 43, 24.56}
	A3 = A{"   Hello   ", true, 42, 23.45}
	E1 = A{"", true, 42, 23.45}
	E2 = A{"Hello", false, 42, 23.45}
	E3 = A{"Hello", true, 0, 23.45}
	E4 = A{"Hello", true, 42, 0.0}
	E5 = A{"", false, 0, 0.0}
	X1 = B{"Hello", true, 42, 23.45, map[string]string{"x": "X", "y": "Y"}}
)

func CheckA(t *testing.T, a *A, b A) {
	if a.String != b.String {
		t.Errorf("invalid string got=%s expected=%s", a.String, b.String)
	}
	if a.Bool != b.Bool {
		t.Errorf("invalid bool got=%v expected=%v", a.Bool, b.Bool)
	}
	if a.Int != b.Int {
		t.Errorf("invalid int got=%d expected=%d", a.Int, b.Int)
	}
	if a.Float != b.Float {
		t.Errorf("invalid float got=%f expected=%f", a.Float, b.Float)
	}
}

func CheckB(t *testing.T, a *B, b B) {
	if a.String != b.String {
		t.Errorf("invalid string got=%s expected=%s", a.String, b.String)
	}
	if a.Bool != b.Bool {
		t.Errorf("invalid bool got=%v expected=%v", a.Bool, b.Bool)
	}
	if a.Int != b.Int {
		t.Errorf("invalid int got=%d expected=%d", a.Int, b.Int)
	}
	if a.Float != b.Float {
		t.Errorf("invalid float got=%f expected=%f", a.Float, b.Float)
	}
	if (a.Any == nil) != (b.Any == nil) {
		t.Errorf("invalid map got=%s expected=%s", a.Any, b.Any)
	}
	if la, lb := len(a.Any), len(b.Any); la != lb {
		t.Errorf("invalid map size got=%d expected=%d", la, lb)
	}
	if a.Any != nil && b.Any != nil {
		for n, v := range b.Any {
			vv, ok := a.Any[n]
			if !ok {
				t.Errorf("missing map entry %s", n)
				continue
			}
			if v != vv {
				t.Errorf("invalid map entry %s, got=%s expected=%s", n, vv, v)
			}
		}
	}
}

func CheckC(t *testing.T, a *C, b B) {
	if a.String != b.String {
		t.Errorf("invalid string got=%s expected=%s", a.String, b.String)
	}
	if a.Bool != b.Bool {
		t.Errorf("invalid bool got=%v expected=%v", a.Bool, b.Bool)
	}
	if a.Int != b.Int {
		t.Errorf("invalid int got=%d expected=%d", a.Int, b.Int)
	}
	if a.Float != b.Float {
		t.Errorf("invalid float got=%f expected=%f", a.Float, b.Float)
	}
	if (a.Any == nil) != (b.Any == nil) {
		t.Errorf("invalid map got=%s expected=%s", a.Any, b.Any)
	}
	if la, lb := len(a.Any), len(b.Any); la != lb {
		t.Errorf("invalid map size got=%d expected=%d", la, lb)
	}
	if a.Any != nil && b.Any != nil {
		for n, v := range b.Any {
			vv, ok := a.Any[n]
			if !ok {
				t.Errorf("missing map entry %s", n)
				continue
			}
			if v != vv.String() {
				t.Errorf("invalid map entry %s, got=%s expected=%s", n, vv, v)
			}
		}
	}
}

func CheckD(t *testing.T, a *D, b B) {
	if a.String != b.String {
		t.Errorf("invalid string got=%s expected=%s", a.String, b.String)
	}
	if a.Bool != b.Bool {
		t.Errorf("invalid bool got=%v expected=%v", a.Bool, b.Bool)
	}
	if a.Int != b.Int {
		t.Errorf("invalid int got=%d expected=%d", a.Int, b.Int)
	}
	if a.Float != b.Float {
		t.Errorf("invalid float got=%f expected=%f", a.Float, b.Float)
	}
	if (a.Any == nil) != (b.Any == nil) {
		t.Errorf("invalid map got=%s expected=%s", a.Any, b.Any)
	}
	if la, lb := len(a.Any), len(b.Any); la != lb {
		t.Errorf("invalid map size got=%d expected=%d", la, lb)
	}
	if a.Any != nil && b.Any != nil {
		for n, v := range b.Any {
			vv, ok := a.Any[n]
			if !ok {
				t.Errorf("missing map entry %s", n)
				continue
			}
			if v != vv.String() {
				t.Errorf("invalid map entry %s, got=%s expected=%s", n, vv, v)
			}
		}
	}
}

func CheckMap(t *testing.T, m map[string]string, b A) {
	var err error
	a := &A{}
	if s, ok := m["s"]; !ok {
		t.Errorf("missing map entry 's'")
	} else {
		a.String = s
	}
	if s, ok := m["b"]; !ok {
		t.Errorf("missing map entry 'b'")
	} else {
		if a.Bool, err = strconv.ParseBool(s); err != nil {
			t.Error(err)
		}
	}
	if s, ok := m["i"]; !ok {
		t.Errorf("missing map entry 'i'")
	} else {
		if a.Int, err = strconv.ParseInt(s, 10, 64); err != nil {
			t.Error(err)
		}
	}
	if s, ok := m["f"]; !ok {
		t.Errorf("missing map entry 'f'")
	} else {
		if a.Float, err = strconv.ParseFloat(s, 64); err != nil {
			t.Error(err)
		}
	}
	// check contents
	CheckA(t, a, b)
}

func TestUnmarshalFromByte(t *testing.T) {
	a := make([]*A, 0)
	if err := Unmarshal([]byte(CsvWithHeader), &a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalFromReader(t *testing.T) {
	r := bytes.NewReader([]byte(CsvWithHeader))
	dec := NewDecoder(r)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalNoSlice(t *testing.T) {
	v := &A{}
	if err := Unmarshal([]byte(CsvWithHeader), v); err == nil {
		t.Errorf("expected error when calling without slice")
	}
}

func TestUnmarshalNoPtr(t *testing.T) {
	v := A{}
	if err := Unmarshal([]byte(CsvWithHeader), v); err == nil {
		t.Errorf("expected error when calling without pointer")
	}
}

func TestUnmarshalWithoutHeader(t *testing.T) {
	r := bytes.NewReader([]byte(CsvWithoutHeader))
	dec := NewDecoder(r).Header(false)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalQuoted(t *testing.T) {
	r := bytes.NewReader([]byte(CsvQuoted))
	dec := NewDecoder(r).Header(false)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalRecords(t *testing.T) {
	r := bytes.NewReader([]byte(CsvWithHeader))
	dec := NewDecoder(r)
	line, err := dec.ReadLine()
	if err != nil {
		t.Error(err)
		return
	}
	if _, err = dec.DecodeHeader(line); err != nil {
		t.Error(err)
		return
	}
	line, err = dec.ReadLine()
	if err != nil && err != io.EOF {
		t.Error(err)
		return
	}
	a := &A{}
	if err = dec.DecodeRecord(a, line); err != nil {
		t.Error(err)
		return
	}
	CheckA(t, a, A1)
}

func TestUnmarshalWithTrim(t *testing.T) {
	r := bytes.NewReader([]byte(CsvWhitespace))
	dec := NewDecoder(r).Header(false).Trim(true)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalWithSeparator(t *testing.T) {
	r := bytes.NewReader([]byte(CsvSemicolon))
	dec := NewDecoder(r).Header(false).Separator(';')
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalWithComments(t *testing.T) {
	r := bytes.NewReader([]byte(CsvComment))
	dec := NewDecoder(r).Header(false)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 2 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 2)
		return
	}
	CheckA(t, a[0], A1)
	CheckA(t, a[1], A2)
}

func TestUnmarshalEmptyLines(t *testing.T) {
	r := bytes.NewReader([]byte(CsvEmptyLine))
	dec := NewDecoder(r).Header(false)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 2 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 2)
		return
	}
	CheckA(t, a[0], A1)
	CheckA(t, a[1], A2)
}

func TestUnmarshalEmptyFields(t *testing.T) {
	r := bytes.NewReader([]byte(CsvEmptyField))
	dec := NewDecoder(r).Header(false)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 5 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 5)
		return
	}
	CheckA(t, a[0], E1)
	CheckA(t, a[1], E2)
	CheckA(t, a[2], E3)
	CheckA(t, a[3], E4)
	CheckA(t, a[4], E5)
}

func TestUnmarshalDontSkipUnknown(t *testing.T) {
	r := bytes.NewReader([]byte(CsvUnknownField))
	dec := NewDecoder(r).SkipUnknown(false)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err == nil {
		t.Errorf("expected error when not ignoring unknown fields")
	}
}

func TestUnmarshalSkipUnknown(t *testing.T) {
	r := bytes.NewReader([]byte(CsvUnknownField))
	dec := NewDecoder(r).SkipUnknown(true)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 1)
		return
	}
	CheckA(t, a[0], A1)
}

func TestUnmarshalToMap(t *testing.T) {
	m := make(map[string]string)
	r := bytes.NewReader([]byte(CsvWithHeader))
	dec := NewDecoder(r)
	line, err := dec.ReadLine()
	if err != nil {
		t.Error(err)
		return
	}
	if _, err = dec.DecodeHeader(line); err != nil {
		t.Error(err)
		return
	}
	line, err = dec.ReadLine()
	if err != nil && err != io.EOF {
		t.Error(err)
		return
	}
	if err = dec.DecodeRecord(&m, line); err != nil {
		t.Error(err)
		return
	}
	if len(m) != 4 {
		t.Errorf("invalid field count, got=%d expected=%d", len(m), 4)
		return
	}
	CheckMap(t, m, A1)
}

func TestUnmarshalAny(t *testing.T) {
	r := bytes.NewReader([]byte(CsvAnyFields))
	dec := NewDecoder(r)
	b := make([]*B, 0)
	if err := dec.Decode(&b); err != nil {
		t.Error(err)
	}
	if len(b) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(b), 1)
		return
	}
	CheckB(t, b[0], X1)
}

func TestUnmarshalAnyTextMarshaler(t *testing.T) {
	r := bytes.NewReader([]byte(CsvAnyFields))
	dec := NewDecoder(r)
	c := make([]*C, 0)
	if err := dec.Decode(&c); err != nil {
		t.Error(err)
	}
	if len(c) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(c), 1)
		return
	}
	CheckC(t, c[0], X1)
}

func TestUnmarshalAnyStructPtrTextMarshaler(t *testing.T) {
	r := bytes.NewReader([]byte(CsvAnyFields))
	dec := NewDecoder(r)
	d := make([]*D, 0)
	if err := dec.Decode(&d); err != nil {
		t.Error(err)
	}
	if len(d) != 1 {
		t.Errorf("invalid record count, got=%d expected=%d", len(d), 1)
		return
	}
	CheckD(t, d[0], X1)
}

func TestUnmarshalCRLF(t *testing.T) {
	r := bytes.NewReader([]byte(CsvWithCRLF))
	dec := NewDecoder(r)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 2 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 2)
		return
	}
	CheckA(t, a[0], A1)
	CheckA(t, a[1], A2)
}

func TestLastRecordWithoutLF(t *testing.T) {
	r := bytes.NewReader([]byte(CsvWithoutLF))
	dec := NewDecoder(r)
	a := make([]*A, 0)
	if err := dec.Decode(&a); err != nil {
		t.Error(err)
	}
	if len(a) != 2 {
		t.Errorf("invalid record count, got=%d expected=%d", len(a), 2)
		return
	}
	CheckA(t, a[0], A1)
	CheckA(t, a[1], A2)
}
