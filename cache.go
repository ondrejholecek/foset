// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

// Experimental cache support.
//
// Filters can be applied on the original plain text file and results
// can be stored in the cache file.
// Further reads can be performed from this cache file with increased
// speed.

package main

import (
	hprose "github.com/hprose/hprose-golang/io"
	"encoding/binary"
	"compress/gzip"
	"sync"
	"strings"
	"os"
	"math/rand"
	"fmt"
	"foset/fortisession"
	"io"
)

type CacheFile struct {
	writer []*gzip.Writer
	reader []*gzip.Reader
	lock   []*sync.Mutex
	parts     int
}

func CacheInit(filename string, mode string, parts int) (*CacheFile, error) {
	var c     CacheFile
	var file  *os.File
	var err   error = nil

	c.writer = make([]*gzip.Writer, parts)
	c.reader = make([]*gzip.Reader, parts)
	c.lock   = make([]*sync.Mutex, parts)

	c.parts = parts
	for part := 0; part < c.parts; part++ {
		partname := fmt.Sprintf("%s.%d", filename, part)
		if strings.Contains(mode, "w") {
			file, err = os.OpenFile(partname, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0644)
			if err == nil {
				c.writer[part], err = gzip.NewWriterLevel(file, gzip.BestSpeed)
			}
		} else {
			file, err = os.OpenFile(partname, os.O_RDONLY, 0)
			if err == nil {
				c.reader[part], err = gzip.NewReader(file)
			}
		}

		if err != nil {
			err = fmt.Errorf("Unable to open cache file (mode \"%s\"): %s", mode, err)
			break
		}

		c.lock[part] = &sync.Mutex{}
	}

	return &c, err
}

func (c *CacheFile) Finalize() {
	for part := 0; part < c.parts; part++ {
		c.lock[part].Lock()
		if c.writer[part] != nil { c.writer[part].Close() }
		if c.reader[part] != nil { c.reader[part].Close() }
		c.lock[part].Unlock()
	}
}

func (c *CacheFile) Write(session *fortisession.Session) (error) {
	data := hprose.Serialize(session, true)

	b := make([]byte, 8)
	binary.LittleEndian.PutUint64(b, uint64(len(data)))

	var err error
	part := rand.Intn(c.parts)
	c.lock[part].Lock()
	err = write_exactly(c.writer[part], b)
	if err == nil { err = write_exactly(c.writer[part], data) }
	c.lock[part].Unlock()

	if err != nil {
		return fmt.Errorf("Unable to serialize to cache file: %s", err)
	} else {
		return nil
	}
}

func (c *CacheFile) ReadAll(sessions chan *fortisession.Session) (error) {
	done := make(chan error, c.parts)

	for part := 0; part < c.parts; part++ {
		go c.read_part(part, sessions, done)
	}

	// wait for all gorutines to finish
	var err error = nil
	for part := 0; part < c.parts; part++ {
		tmp := <-done
		if tmp != nil { err = tmp }
	}

	return err
}

func (c *CacheFile) read_part(part int, sessions chan *fortisession.Session, done chan error) {
	b := make([]byte, 8)
	var err error

	for err != io.EOF {
//		c.lock.Lock()

		err = read_exactly(c.reader[part], b)
		length := int(binary.LittleEndian.Uint64(b))

		data := make ([]byte, length)
		err = read_exactly(c.reader[part], data)

//		c.lock.Unlock()
		if err != nil && err != io.EOF { break }

		var session fortisession.Session
		hprose.Unserialize(data, &session, true)
		sessions <- &session
	}

	if err != nil && err != io.EOF {
		done <- fmt.Errorf("Unable to deserialize from cache file (part %d): %s", part, err)
	} else {
		done <- nil
	}
}

func read_exactly(reader *gzip.Reader, buf []byte) (error) {
	var r, total int
	var err error

	total = 0
	for total < len(buf) {
		r, err = reader.Read(buf[total:])
		if err != nil { break }

		total += r
	}

	return err
}

func write_exactly(writer *gzip.Writer, buf []byte) (error) {
	var w, total int
	var err error

	total = 0
	for total < len(buf) {
		w, err = writer.Write(buf[total:])
		if err != nil { break }

		total += w
	}

	return err
}
