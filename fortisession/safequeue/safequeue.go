// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package safequeue

import (
	"container/list"
	"sync"
	"time"
	"github.com/juju/loggo"
)

var log loggo.Logger

type SafeQueue struct {
	plains *list.List
	lock   *sync.Mutex
	active bool
}

func Init(custom_log loggo.Logger) (*SafeQueue) {
	log = custom_log

	var p SafeQueue
	p.plains = list.New()
	p.lock = &sync.Mutex{}
	p.active = true
	return &p
}

func (p *SafeQueue) Push(plains *list.List) {
	for p.plains.Len() > 32768 {
		time.Sleep(100 * time.Millisecond)
	}
	p.lock.Lock()
	p.plains.PushBackList(plains)
	p.lock.Unlock()
}

func (p *SafeQueue) Pop(size int) ([][]byte) {
	plains := make([][]byte, 0)

	p.lock.Lock()
	e := p.plains.Front()

	for i := 0; i<size; i++ {
		if e == nil { break }

		plains = append(plains, e.Value.([]byte))
		n := e.Next()
		p.plains.Remove(e)
		e = n
	}

	p.lock.Unlock()
	return plains
}

func (p *SafeQueue) IsActive() bool {
	return p.active
}

func (p *SafeQueue) IsEmpty() bool {
	if p.plains.Len() > 0 {
		return false
	} else {
		return true
	}
}

func (p *SafeQueue) Finish() {
	// Need first to wait for the queue to become empty and then deactivate processors
	// .. if we deactivate it first, it may never become empty..

	for !p.IsEmpty() {
		time.Sleep(10 * time.Millisecond)
	}

	p.active = false
}


