go-csv
===========

[![GoDoc](https://godoc.org/github.com/trimmer-io/go-csv/timecode?status.svg)](https://godoc.org/github.com/trimmer-io/go-csv)


go-csv is a [Go](http://golang.org/) package for encoding and decoding CSV-structured textfiles to/from arbitrary Go types.

Features
--------

- [RFC 4180](https://tools.ietf.org/rfc/rfc4180.txt) compliant CSV reader and writer
- MarshalCSV/UnmarshalCSV interfaces
- mapping to strings, integers, floats and boolean values
- bulk or stream processing
- custom separator and comment characters
- optional whitespace trimming for headers and string values
- `any` support for reading unknown CSV fields

### TODO

- parse quoted strings containing newline and comma
- quote strings containing comma, newline and double-quotes on output

Documentation
-------------

- [API Reference](http://godoc.org/github.com/trimmer-io/go-csv)
- [FAQ](https://github.com/github.com/trimmer-io/go-csv/wiki/FAQ)

Installation
------------

Install go-csv using the "go get" command:

    go get github.com/trimmer-io/go-csv

The Go distribution is go-csv's only dependency.

Examples
--------

### Reading a well defined CSV file

This example assumes your CSV file contains a header who's values match the struct tags defined on the Go type FrameInfo. CSV fields that are undefined in the type are ignored.

```
import "github.com/trimmer-io/go-csv"

type FrameInfo struct {
	ActiveImageHeight  int      `csv:"Active Image Height"`
	ActiveImageLeft    int      `csv:"Active Image Left"`
	ActiveImageTop     int      `csv:"Active Image Top"`
	ActiveImageWidth   int      `csv:"Active Image Width"`
	CameraClipName     string   `csv:"Camera Clip Name"`
	CameraRoll         float32  `csv:"Camera Roll"`
	CameraTilt         float32  `csv:"Camera Tilt"`
	MasterTC           string   `csv:"Master TC"`
	MasterTCFrameCount int      `csv:"Master TC Frame Count"`
	SensorFPS          float32  `csv:"Sensor FPS"`
}

type FrameSequence []*FrameInfo

func ReadFile(path string) (FrameSequence, error) {
	b, err := ioutil.ReadFile(path)
	if err != nil {
		return nil, err
    	}
	seq := make(FrameSequence, 0)
	if err := csv.Unmarshal(b, &seq); err != nil {
		return nil, err
	}
	return seq, nil
}
```

### Fail when encountering unknown CSV fields
```
func ReadFileUnknown(path string) (FrameSequence, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
    	defer f.Close()
	dec := csv.NewDecoder(f).SkipUnknown(false)
	c := make(FrameSequence, 0)
	if err := dec.Decode(&c); err != nil {
		return nil, err
    	}
	return c, nil
}
```

### Parsing an unknown CSV file into a slice of maps
```
type GenericRecord struct {
	Record map[string]string `csv:,any`
}

type GenericCSV []GenericRecord

func ReadFileIntoMap(path string) (GenericCSV, error) {
	f, err := os.Open(path)
	if err != nil {
	    return nil, err
    	}
    	defer f.Close()
	dec := csv.NewDecoder(f)
	c := make(GenericCSV, 0)
	if err := dec.Decode(&c); err != nil {
	    return nil, err
    	}
	return c, nil
}
```

### Stream-process CSV input
```
func ReadStream(r io.Reader) error {
	dec := csv.NewDecoder(r)

	// read and decode the file header
	line, err := dec.ReadLine()
	if err != nil {
		return err
	}
	if _, err = dec.DecodeHeader(line); err != nil {
		return err
	}

	// loop until EOF (i.e. dec.ReadLine returns an empty line and nil error);
	// any other error during read will result in a non-nil error
	for {
		// read the next line from stream
		line, err = dec.ReadLine()

		// check for read errors other than EOF
		if err != nil {
			return err
		}

		// check for EOF condition
		if line == "" {
			break
		}

		// decode the record
		v := &FrameInfo{}
		if err = dec.DecodeRecord(v, line); err != nil {
			return err
		}

		// process the record here
		Process(v)
	}
	return nil
}
```


Contributing
------------

See [CONTRIBUTING.md](https://github.com/trimmer-io/go-csv/blob/master/.github/CONTRIBUTING.md).


License
-------

go-csv is available under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).

