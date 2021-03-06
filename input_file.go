// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package main

import (
	"fmt"
	"io"
	"bufio"
	"bytes"
	"time"
	"sync/atomic"
	"container/list"
	"compress/gzip"
	"foset/iproviders/common"
	"foset/fortisession"
	"foset/fortisession/safequeue"
	"foset/plugins/common"
	"foset/fortisession/forticonditioner"
)

type Compression struct {
	Gzip bool
}

type FileProcessing struct {
	threads  int
	done     chan bool
	sq       *safequeue.SafeQueue
	progress       io.Writer
	progressParams *iprovider_common.WriterParams
	rd_total uint64
	rd_match uint64
}


//
func (fp *FileProcessing) process_sessions(results chan *fortisession.Session, req *fortisession.SessionDataRequest, done chan bool, conditioner *forticonditioner.Condition, plugins []*plugin_common.FosetPlugin) {

	for fp.sq.IsActive() {
		count := 0
		for _, plain := range fp.sq.Pop(128) {
			session := fortisession.Parse(plain, req)

			// count all parsed sessions
			atomic.AddUint64(&fp.rd_total, 1)
			if fp.progress != nil {
				fp.progress.Write([]byte(fmt.Sprintf("SFRS:%d\n", fp.rd_total)))
			}

			if run_plugins(plugins, PLUGINS_BEFORE_FILTER, session) { continue }
			if conditioner != nil && !conditioner.Matches(session) { continue }

			// count sessions matching filter
			atomic.AddUint64(&fp.rd_match, 1)
			if fp.progress != nil {
				fp.progress.Write([]byte(fmt.Sprintf("SFMS:%d\n", fp.rd_match)))
			}

			if run_plugins(plugins, PLUGINS_AFTER_FILTER, session) { continue }

			results <- session
			count += 1
		}

		if count == 0 {
			time.Sleep(10 * time.Millisecond)
		}
	}

	done <- true
}

func scanner_split(data []byte, atEOF bool) (advance int, token []byte, err error) {
	log.Tracef("scanner_split data: %s\n---\n", string(data))
	// are we at the start of a new session?
	// if yes (i==0) ok, do the rest
	// else either ask for bigger buffer (i==-1)
	// or shift to the start that we have just found (i>0)
	session_start_data := []byte("\nsession info:")
	i := bytes.Index(data, session_start_data)
	if i == -1 {
		if len(session_start_data) > len(data) {
			return 0, nil, nil
		} else {
			return len(data)-len(session_start_data), nil, nil
		}
	} else if i > 0  {
		return i, nil, nil
	}

	// to find the end of this session data, easiest way it to find the beginning of the next session data
	n := bytes.Index(data[1:], session_start_data)
	if n == -1 {
		// if we don't have next session info, it can mean two things:
		// 1) the buffer we are inspecting is not big enough - return 0,nil,nil to signalize it to Scanner
		// 2) this is really the last session - in that case we should see the empty line
		nl := bytes.Index(data[1:], []byte("\ntotal session"))
		if nl == -1 { nl = bytes.Index(data[1:], []byte("\n\n"))   }
		if nl == -1 { nl = bytes.Index(data[1:], []byte("\n\r\n")) }

		if nl == -1 && atEOF {
			return len(data), data, nil
		} else if nl == -1 {
			return 0, nil, nil
		} else {
			return 1+nl, data[:nl], nil
		}
	} else if n > 0 {
		// find start of the next session, return this one and shift pointer to the start of next one
		//return 1+n, data[i+1 : n], nil
		return 1+n, data[i : n], nil
	} else {
		return 0, nil, nil  // should probably return some error here
	}
}

func Init_file_processing(results chan *fortisession.Session, req *fortisession.SessionDataRequest, threads int, conditioner *forticonditioner.Condition, plugins []*plugin_common.FosetPlugin, progfilename string) (*FileProcessing) {
	done := make(chan bool, threads)

	fp := FileProcessing {
		threads  : threads,
		done     : done,
		sq       : safequeue.Init(log.Child("safequeue")),
	}

	// if progress file name is specified, open it
	if progfilename != "" {
		var err error
		fp.progress, fp.progressParams, err = inputs.ProvideBufferedWriter(progfilename)
		if err != nil { log.Errorf("Cannot open progress file: %s", err) }
	}

	for i := 0; i < threads; i++ {
		go fp.process_sessions(results, req, done, conditioner, plugins)
	}

	return &fp
}

func (fp *FileProcessing) Read_all_from_file(filename string, compression Compression) (error) {
	// where to read the data from 
	var reader  io.Reader
	var creader *CountingReader
	var err     error

	// use input provider
	reader, _, err = inputs.ProvideReader(filename)
	if err != nil { return fmt.Errorf("cannot read session data: %s", err) }
	creader = CountingReaderInit(reader)

	// is the input somehow compressed?
	if compression.Gzip {
		tmp, err := gzip.NewReader(creader)
		if err != nil {
			return fmt.Errorf("Source stream is not gzip compressed: %s", err)
		}
		reader = tmp
	} else {
		reader = creader
	}

	// add new line at the beggining
	// This is a little workaround because the scanner_split function
	// expects the string "session info:" to be preceded by a new line.
	// This worked well unless the "session info:" was at the very
	// beggining of the file. It also didn't work for copy & paste
	// session on stdin.
	newline := bytes.NewReader([]byte("\n"))
	multireader := io.MultiReader(newline, reader)

	scanner := bufio.NewScanner(multireader)
	scanner.Split(scanner_split)
	buf := list.New()

	// read the whole file, split it by session paragraphs and push those to "processing" queue
	// gorutinesprocess_sessions will paralelly retrive that, convert to Session and push 
	// "results" channel
	var last_progress int
	for scanner.Scan() {
		// save progress if requested
		if fp.progress != nil && last_progress != creader.BytesRead {
			fp.progress.Write([]byte(fmt.Sprintf("SFRB:%d\n", creader.BytesRead)))
			last_progress = creader.BytesRead
		}

		session := make([]byte, len(scanner.Bytes()))
		copy(session, scanner.Bytes())
		log.Tracef("Read session:\n%s\n---end---\n", session)
		buf.PushBack(session)
		if buf.Len() >= 1024 {
			fp.sq.Push(buf)
			buf = list.New()
		}
	}
	if buf.Len() > 0 { fp.sq.Push(buf) }

	// Finish will wait for queue to get empty and them will deactivate it
	fp.sq.Finish()
	// and wait for all the workers to finish
	for i := 0; i < fp.threads; i++ {
		<-fp.done
	}
	// flush progress file
	if fp.progress != nil && fp.progressParams.Buffered != nil {
		fp.progressParams.Buffered.Flush()
	}

	return nil
}
