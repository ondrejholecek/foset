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
	"foset/fortisession"
	"foset/fortisession/safequeue"
	"foset/plugins/common"
	"foset/fortisession/forticonditioner"
)

//
func process_sessions(sq *safequeue.SafeQueue, results chan *fortisession.Session, req *fortisession.SessionDataRequest, done chan bool, total_count *uint64, conditioner *forticonditioner.Condition, plugins []*plugin_common.FosetPlugin) {

	for sq.IsActive() {
		count := 0
		for _, plain := range sq.Pop(128) {
			session := fortisession.Parse(plain, req)

			atomic.AddUint64(total_count, 1)
			if *total_count % 100000 == 0 {
				log.Debugf("Processed %d sessions", *total_count)
			}

			if run_plugins(plugins, PLUGINS_BEFORE_FILTER, session) { continue }
			if conditioner != nil && !conditioner.Matches(session) { continue }
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


//

type Compression struct {
	Gzip bool
}

type FileProcessing struct {
	threads int
	done    chan bool
	sq      *safequeue.SafeQueue
}

func Init_file_processing(results chan *fortisession.Session, req *fortisession.SessionDataRequest, threads int, conditioner *forticonditioner.Condition, plugins []*plugin_common.FosetPlugin) (*FileProcessing) {
	done := make(chan bool, threads)
	var count uint64

	fp := FileProcessing {
		threads: threads,
		done   : done,
		sq     : safequeue.Init(log.Child("safequeue")),
	}

	for i := 0; i < threads; i++ {
		go process_sessions(fp.sq, results, req, done, &count, conditioner, plugins)
	}

	return &fp
}

func (state *FileProcessing) Read_all_from_file(filename string, compression Compression) (error) {
	// where to read the data from 
	var reader io.Reader
	var err    error

	// use input provider
	reader, err = inputs.Provide(filename)
	if err != nil { return fmt.Errorf("cannot read session data: %s", err) }

	// is the input somehow compressed?
	if compression.Gzip {
		tmp, err := gzip.NewReader(reader)
		if err != nil {
			return fmt.Errorf("Source stream is not gzip compressed: %s", err)
		}
		reader = tmp
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
	for scanner.Scan() {
		session := make([]byte, len(scanner.Bytes()))
		copy(session, scanner.Bytes())
		log.Tracef("Read session:\n%s\n---end---\n", session)
		buf.PushBack(session)
		if buf.Len() >= 1024 {
			state.sq.Push(buf)
			buf = list.New()
		}
	}
	if buf.Len() > 0 { state.sq.Push(buf) }

	// Finish will wait for queue to get empty and them will deactivate it
	state.sq.Finish()
	// and wait for all the workers to finish
	for i := 0; i < state.threads; i++ {
		<-state.done
	}

	return nil
}
