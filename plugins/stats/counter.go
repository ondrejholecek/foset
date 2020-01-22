// Counter struct is quite generic counter whose `key` can be any object (interface{} that can be map index),
// and `value` is uint64 counter that can be increased by any value.
//
// The writer function is supposed to be called at the very end. It can do anything, but the idea here
// is to dump keys and values as JavaScript variables to the file description that has been passed in.

package plugin_stats

import (
	"io"
	"sync"
)

// Element holds one element of the counter. That is a single key with its count.
type Element struct {
	obj     interface{}
	counter uint64
}

// Counter holds all elements of the counter and also implements the Sort() interface.
type Counter struct {
	mutex   sync.Mutex
	name    string

	elems   []*Element
	obj2c   map[interface{}]*Element

	writer  func(*Counter, io.Writer, map[string]interface{})

	sort_by_value  bool
	sort_low_high  bool
}

// CounterInit creates a new counter.
// Parameter `name` must be something that can be used as single variable name in JavaScript.
// Parameter `writer` is a function that will be called to dump the counter to JavaScript.
func CounterInit(name string, writer func(*Counter, io.Writer, map[string]interface{})) (*Counter) {
	var c Counter
	c.name    = name
	c.elems   = make([]*Element, 0)
	c.obj2c   = make(map[interface{}]*Element, 0)
	c.writer  = writer

	c.sort_by_value  = true
	c.sort_low_high  = true

	return &c
}

// By default the counter is sorted by its value when the Sort() function is called. For some specific counters
// it may be wanted to sort them by key.
func (c *Counter) SortedByKey() {
	c.sort_by_value = false
}

// Normally during the Sort() function the counter is sorted from low to high (0 -> 9999...). It some cases it
// might be useful to sort in a reverse order (9999... -> 0).
func (c *Counter) SortedHighToLow() {
	c.sort_low_high = false
}

// Len returns the number of elements (hashes) in the counter.
func (c *Counter) Len() int {
	return len(c.elems)
}

// Less implements interface for Sort().
func (c *Counter) Less(a,b int) bool {
	var r bool

	if c.sort_by_value {
		r = c.elems[a].counter < c.elems[b].counter
	} else {
		r = c.elems[a].obj.(uint64) < c.elems[b].obj.(uint64)
	}

	if c.sort_low_high {
		return r
	} else {
		return !r
	}
}

// Swap implements interface for Sort().
func (c *Counter) Swap(a,b int) {
	c.mutex.Lock()
	c.elems[a], c.elems[b] = c.elems[b], c.elems[a]
	c.mutex.Unlock()
}

// Add increases the counter value for the specified object by the specified number.
func (c *Counter) Add(obj interface{}, counter uint64) {
	c.mutex.Lock()
	p, e := c.obj2c[obj]

	if !e {
		elems := Element{obj: obj, counter: counter}
		c.elems = append(c.elems, &elems)
		c.obj2c[obj] = &elems

	} else {
		p.counter += counter
	}

	c.mutex.Unlock()
}

// AddOne calls Add with 1.
func (c *Counter) AddOne(obj interface{}) {
	c.Add(obj, 1)
}

// GetRevIndex return the element that is Xth from last. By default the counters are sorted from 0 to 999...,
// hence the last element (this index 0) is the biggest one.
// Returns the object which needs to be asserted to specific type from interface type and its counter.
func (c *Counter) GetRevIndex(position_from_end int) (interface{}, uint64) {
	c.mutex.Lock()
	position_from_end = len(c.elems)-position_from_end-1
	a, b := c.elems[position_from_end].obj, c.elems[position_from_end].counter
	c.mutex.Unlock()
	return a, b
}

// Write data calls the writer function for this counter.
func (c *Counter) WriteData(w io.Writer, params map[string]interface{}) {
	if c.writer == nil { return }
	c.writer(c, w, params)
}

