go-csv
===========

[![Build Status](https://travis-ci.org/echa/go-csv.svg?branch=master)](https://travis-ci.org/echa/go-csv)
[![GoDoc](https://godoc.org/github.com/echa/go-csv/timecode?status.svg)](https://godoc.org/github.com/echa/go-csv)


go-csv is a [Go](http://golang.org/) package for encoding and decoding CSV-structured textfiles to/from arbitrary Go types.

Features
--------

- [RFC 4180](https://tools.ietf.org/rfc/rfc4180.txt) compliant CSV reader and writer
- MarshalCSV()/UnmarshalCSV() interfaces
- mapping to strings, integers, floats and boolean values
- bulk or stream processing
- custom separator and comment characters
- optional whitespace trimming for headers and string values
- `any` support for reading unknown CSV fields

TODO
----

- parse quoted strings containing newline and comma
- quote strings containing comma, newline and double-quotes on output

Documentation
-------------

- [API Reference](http://godoc.org/github.com/echa/go-csv)
- [FAQ](https://github.com/echa/go-csv/wiki/FAQ)

Installation
------------

Install go-csv using the "go get" command:

    go get github.com/echa/go-csv

The Go distribution is go-csv's only dependency.

Examples
--------

```
import "github.com/echa/go-csv"


```


Contributing
------------

See [CONTRIBUTING.md](https://github.com/echa/go-csv/blob/master/.github/CONTRIBUTING.md).


License
-------

go-csv is available under the [Apache License, Version 2.0](http://www.apache.org/licenses/LICENSE-2.0.html).

