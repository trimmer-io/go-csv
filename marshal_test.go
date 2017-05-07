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
	"testing"
)

var LF = []byte{'\n'}

const (
	CsvWithHeaderOut    = "s,b,i,f\nHello,true,42,23.45\n"
	CsvWithoutHeaderOut = "Hello,true,42,23.45\n"
	CsvSemicolonOut     = "Hello;true;42;23.45\n"
)

func CheckOutput(t *testing.T, b []byte, s string) {
	if len(b) == 0 {
		t.Errorf("invalid empty output")
	}
	if bytes.HasSuffix(b, LF) && !bytes.HasSuffix([]byte(s), LF) {
		b = bytes.TrimSuffix(b, LF)
	}
	if bytes.Compare(b, []byte(s)) != 0 {
		t.Errorf("invalid output='%s' expected='%s'", string(b), s)
	}
}

func TestMarshalToByte(t *testing.T) {
	a := []A{A1}
	b, err := Marshal(a)
	if err != nil {
		t.Error(err)
	}
	CheckOutput(t, b, CsvWithHeaderOut)
}

func TestMarshalPtrToByte(t *testing.T) {
	a := []*A{&A1}
	b, err := Marshal(a)
	if err != nil {
		t.Error(err)
	}
	CheckOutput(t, b, CsvWithHeaderOut)
}

func TestMarshalToWriter(t *testing.T) {
	var w bytes.Buffer
	enc := NewEncoder(&w)
	a := []*A{&A1}
	if err := enc.Encode(&a); err != nil {
		t.Error(err)
	}
	CheckOutput(t, w.Bytes(), CsvWithHeaderOut)
}

func TestMarshalNoSlice(t *testing.T) {
	_, err := Marshal(A1)
	if err == nil {
		t.Errorf("expected error when calling without slice")
	}
}

func TestMarshalWithoutHeader(t *testing.T) {
	var w bytes.Buffer
	enc := NewEncoder(&w).Header(false)
	a := []*A{&A1}
	if err := enc.Encode(&a); err != nil {
		t.Error(err)
	}
	CheckOutput(t, w.Bytes(), CsvWithoutHeaderOut)
}

func TestMarshalRecords(t *testing.T) {
	var w bytes.Buffer
	enc := NewEncoder(&w)
	if err := enc.EncodeHeader(nil, &A1); err != nil {
		t.Error(err)
	}
	if err := enc.EncodeRecord(&A1); err != nil {
		t.Error(err)
	}
	CheckOutput(t, w.Bytes(), CsvWithHeaderOut)
}

func TestMarshalWithTrim(t *testing.T) {
	var w bytes.Buffer
	enc := NewEncoder(&w).Trim(true)
	if err := enc.EncodeHeader(nil, &A3); err != nil {
		t.Error(err)
	}
	if err := enc.EncodeRecord(&A3); err != nil {
		t.Error(err)
	}
	CheckOutput(t, w.Bytes(), CsvWithHeaderOut)
}

func TestMarshalMissingHeader(t *testing.T) {
	var w bytes.Buffer
	enc := NewEncoder(&w)
	if err := enc.EncodeRecord(&A1); err != nil {
		t.Error(err)
	}
	CheckOutput(t, w.Bytes(), CsvWithHeaderOut)
}

func TestMarshalWithSeparator(t *testing.T) {
	var w bytes.Buffer
	enc := NewEncoder(&w).Header(false).Separator(';')
	if err := enc.EncodeRecord(&A1); err != nil {
		t.Error(err)
	}
	CheckOutput(t, w.Bytes(), CsvSemicolonOut)
}
