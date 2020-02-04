package main

import (
	"io"
)

type CountingReader struct {
	reader    io.Reader
	BytesRead int
}

func (r *CountingReader) Read(p []byte) (n int, err error) {
	n, err = r.reader.Read(p)
	r.BytesRead += n
	return n, err
}

func CountingReaderInit(reader io.Reader) (*CountingReader) {
	return &CountingReader {
		reader    : reader,
		BytesRead : 0,
	}
}
