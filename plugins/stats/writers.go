// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package plugin_stats

import (
	"strings"
	"fmt"
	"sort"
	"io"
	"math/rand"
)

func WriteSimpleData(c *Counter, w io.Writer, params map[string]interface{}) {
	// how many top lines to generate
	var top int = params["top"].(int)

	// allow to transform some strings 
	var transform func(interface{})(string)
	if params["transform"] != nil {
		transform = params["transform"].(func(interface{})(string))
	}

	// allow to translate some strings 
	var translate map[string]string
	if params["translate"] != nil {
		translate = params["translate"].(map[string]string)
	}

	// show others field?
	var showOthers bool
	if params["showOthers"] != nil {
		showOthers = params["showOthers"].(bool)
	}

	// show summary?
	var showSummary bool
	if params["showSummary"] != nil {
		showSummary = params["showSummary"].(bool)
	}

	// write descrition
	var description string
	if params["description"] != nil {
		description = params["description"].(string)
	}

	//
	var valueformat string = "number"
	if params["valueformat"] != nil {
		valueformat = params["valueformat"].(string)
	}

	//
	var title string = c.name
	if params["title"] != nil {
		title = params["title"].(string)
	}

	//
	var tab string = "others"
	if params["tab"] != nil {
		tab = params["tab"].(string)
	}

	if params["sortByKey"] != nil {
		if params["sortByKey"].(bool) {
			c.SortedHighToLow()
			c.SortedByKey()
		}
	}

	sort.Sort(c)

	var summary uint64

	var max int = c.Len()
	if top < max { max = top }

	labels := make([]string, max)
	data   := make([]string, max)

	// get top X
	for i := 0; i < max; i++ {
		obj, count := c.GetRevIndex(i)

		// transform the label if desired
		if transform != nil {
			labels[i] = transform(obj)
		} else {
			labels[i] = fmt.Sprintf("%d", obj)
		}

		// translate label if desired
		n, e := translate[labels[i]]
		if e { labels[i] = n }

		//
		data[i]   = fmt.Sprintf("%d", count)
		summary   += count
	}

	// summarize others
	var sumOthers, items uint64

	for i := max; i < c.Len(); i++ {
		_, count := c.GetRevIndex(i)
		sumOthers += count
		summary   += count
		items     += 1
	}

	if showOthers {
		if items > 0 {
			labels = append(labels, fmt.Sprintf("+ %d others", items))
			data   = append(data, fmt.Sprintf("%d", sumOthers))
		}
	}

	// write to the output stream
	fmt.Fprintf(w, "current.data.%s = Object();\n", c.name)
	fmt.Fprintf(w, "current.data.%s.labels = ['%s'];\n", c.name, strings.Join(labels, "' ,'"))
	fmt.Fprintf(w, "current.data.%s.data   = ['%s'];\n", c.name, strings.Join(data, "' ,'"))
	fmt.Fprintf(w, "current.data.%s.desc   = \"%s\";\n", c.name, description)
	fmt.Fprintf(w, "current.data.%s.format = \"%s\";\n", c.name, valueformat)
	fmt.Fprintf(w, "current.data.%s.title  = \"%s\";\n", c.name, title)
	fmt.Fprintf(w, "current.data.%s.tab    = \"%s\";\n", c.name, tab)
	if showSummary {
		fmt.Fprintf(w, "current.data.%s.sum    = %d;\n", c.name, summary)
	}
	fmt.Fprintf(w, "current.order.push(\"%s\");\n", c.name)
}

func WriteSpace(w io.Writer, tab string) {
	var name string = fmt.Sprintf("space_%d", rand.Int63())

	fmt.Fprintf(w, "current.data.%s = Object();\n", name)
	fmt.Fprintf(w, "current.data.%s.tab = \"%s\";\n", name, tab)
	fmt.Fprintf(w, "current.order.push(\"%s\");\n", name)
}
