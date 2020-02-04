package iprovider_common

import (
	"io"
	"bufio"
)

type IProvider interface {
	Name() (string)
	WaitReady() (error)
	CanProvideReader(name string) (bool, int)
	ProvideReader(name string) (io.Reader, *ReaderParams, error)
	CanProvideWriter(name string) (bool, int)
	ProvideWriter(name string) (io.Writer, *WriterParams, error)
}

type WriterParams struct {
	IsTerminal   bool
	Buffered     *bufio.Writer
}

type ReaderParams struct {
	IsTerminal   bool
}

