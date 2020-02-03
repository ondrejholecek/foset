package iprovider_common

import (
	"io"
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
}

type ReaderParams struct {
	IsTerminal   bool
}

