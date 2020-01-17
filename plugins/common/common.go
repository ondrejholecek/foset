// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package plugin_common

import (
	"strings"
	"strconv"
	"fortisession"
)

// Hooks specify the plugin callbacks.
// Hooks BeforeFilter and AfterFilter expect pointer to Session, which they can fully
// access (however, only the `Custom` fieldentries should be added by plugin). 
// Their return boolean specifies
// whether `foset` should ignore the session (true = to not display it) or no (false).
type Hooks struct {
	BeforeFilter     func(*fortisession.Session)(bool)
	AfterFilter      func(*fortisession.Session)(bool)
	Finished         func()
}

// FosetPlugin describes the plugin and specify its usage.
type FosetPlugin struct {
	Hooks   Hooks
}

// ExtractData takes a plugin data string (everything after first "|" i plugin
// specification, splits it to key=value fields divided by coma(s) and return
// three different maps:
// - `known` fields that were passed in in `accepted` parameter and found in data
// - `unknown` text fields that were in data but not in `accepted` parameter
// - `unknown_integers` number fields that were in data but not in `accepted` parameter
// If `defaults` map is not `nil`, the values of `accepted` fields that
// are not found in `data` are taken from `defaults` (is they exist)
func ExtractData(data string, accepted []string, defaults map[string]string) (known map[string]string, unknown map[string]string, unknown_integers map[int]string) {

	// split data fields
	all := make(map[string]string)

	for _, part := range strings.Split(data, ",") {
		var key, value string
		eq := strings.Index(part, "=")
		if eq == -1 {
			key   = part
			value = ""
		} else {
			key   = part[:eq]
			value = part[eq+1:]
		}

		if len(key) > 0 {
			all[key] = value
		}
	}

	// prepare structures
	known   = make(map[string]string)
	unknown = make(map[string]string)
	unknown_integers = make(map[int]string)

	// separate to know, unknown and unknown_integers
	for k, v := range all {
		if stringInArray(k, accepted) {
			known[k] = v
		} else {
			tmp, err := strconv.ParseUint(k, 10, 32)
			if err != nil {
				unknown[k] = v
			} else {
				unknown_integers[int(tmp)] = v
			}
		}
	}

	// make sure all accepted are present in the output
	for _, k := range accepted {
		_, exists := known[k]
		default_value, has_default := defaults[k]
		if !exists && has_default { known[k] = default_value }
	}

	return known, unknown, unknown_integers
}

func stringInArray(search string, array []string) bool {
	for _, v := range array {
		if v == search { return true }
	}
	return false
}
