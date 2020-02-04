// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

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

