// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package iproviders

import (
	"sync"
	"io"
)

type ThreadSafeWriter struct {
	writer   io.Writer
	lock     sync.Mutex
}

func ThreadSafeWriterInit(writer io.Writer) (*ThreadSafeWriter) {
	return &ThreadSafeWriter {
		writer: writer,
	}
}

func (tsw *ThreadSafeWriter) Write(p []byte) (nn int, err error) {
	tsw.lock.Lock()
	nn, err = tsw.writer.Write(p)
	tsw.lock.Unlock()
	return
}
