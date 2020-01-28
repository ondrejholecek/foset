// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package plugin_merge

import (
	"fmt"
	"os"
	"strings"
	"bufio"
	"strconv"
	"fortisession"
	"fortisession/multivalue"
	"foset/plugins/common"
	"github.com/juju/loggo"
)

var log loggo.Logger

var global_data_fields   map[uint64][]*multivalue.MultiValue
var global_data_columns  []string
var global_data_types    []ColumnType
var global_data_defaults []*multivalue.MultiValue

type ColumnType uint8
const (
	CT_STRING        ColumnType = iota
	CT_UINT64_10     ColumnType = iota
	CT_UINT64_16     ColumnType = iota
	CT_FLOAT64       ColumnType = iota
)

func InitPlugin(data string, data_request *fortisession.SessionDataRequest, custom_log loggo.Logger) (*plugin_common.FosetPlugin, error) {
	// setup logging
	log = custom_log.Child("merge")

	// parse data parameters
	defaults := make(map[string]string)
	defaults["sep"] = " "
	defaults["file"] = ""
	defaults["key"] = "serial"
	dk, du, dui := plugin_common.ExtractData(data, []string{"sep","file","key"}, defaults)
//	fmt.Printf("known: %f\nunknown: %f\nunknown integers: %f\n", dk, du, dui)

	// make sure all parameters are correct
	if len(dk["file"]) == 0 {
		return nil, fmt.Errorf("file parameter missing")
	}
	if len(du) > 0 {
		parameter_names := make([]string, 0)
		for k, _ := range du { parameter_names = append(parameter_names, k) }
		return nil, fmt.Errorf("unknown parameter(s): %s", strings.Join(parameter_names, ", "))
	}
	if _, has_index_0 := dui[0]; has_index_0 {
		return nil, fmt.Errorf("fields position numbering should start from 1")
	}

	// find the highest column index
	var max_index int  = -1
	for k, _ := range dui {
		if k > max_index  {  max_index = k   }
	}

	if max_index <= 0 {
		return nil, fmt.Errorf("column names parameters missing")
	}

	// prepare column names and types arrays
	// and save them to globals
	data_columns  := make([]string, max_index)
	data_types    := make([]ColumnType, max_index)
	data_defaults := make([]*multivalue.MultiValue, max_index)
	key_index     := int(-1)

	for i := 1; i <= max_index; i++ {
		value, _ := dui[i]
		perc     := strings.Index(value, "%")

		if perc == -1 {
			data_columns[i-1]  = value
			data_types[i-1]    = CT_STRING
			data_defaults[i-1] = multivalue.NewString("")

		} else {
			data_columns[i-1] = value[:perc]
			tt := value[perc+1:]

			if tt == "s" {
				data_types[i-1]    = CT_STRING
				data_defaults[i-1] = multivalue.NewString("")

			} else if tt == "d" {
				data_types[i-1]    = CT_UINT64_10
				data_defaults[i-1] = multivalue.NewUint64(0)

			} else if tt == "x" {
				data_types[i-1]    = CT_UINT64_16
				data_defaults[i-1] = multivalue.NewUint64(0)

			} else if tt == "f" {
				data_types[i-1]    = CT_FLOAT64
				data_defaults[i-1] = multivalue.NewFloat64(0)

			} else {
				return nil, fmt.Errorf("unknown field type \"%s\" for field \"%s\"", tt, value)
			}
		}

		// find key index
		if data_columns[i-1] == dk["key"] {
			key_index = i-1
		}
	}

	global_data_columns  = data_columns
	global_data_types    = data_types
	global_data_defaults = data_defaults

	// make sure we have key column
	if key_index == -1 {
		return nil, fmt.Errorf("key field name \"%s\" not present in fields", dk["key"])
	}

	// load the data from file and save it to global variable
	data_fields, err := load_file(dk["file"], dk["sep"], key_index)
	if err != nil { return nil, err }
	global_data_fields = data_fields

	// request fields
	data_request.Custom     = true
	data_request.Serial     = true

	// setup callbacks
	var hooks plugin_common.Hooks
	hooks.BeforeFilter = ProcessSession

	var pluginInfo plugin_common.FosetPlugin
	pluginInfo.Hooks = hooks

	//
	return &pluginInfo, nil
}

func ProcessSession(session *fortisession.Session) bool {
	data, found := global_data_fields[session.Serial]

	for i, field := range global_data_columns {
		if len(field) == 0  { continue }

		if !found                 { session.Custom[field] = global_data_defaults[i]
		} else if i >= len(data)  { session.Custom[field] = global_data_defaults[i]
		} else                    { session.Custom[field] = data[i] }
	}

	return false
}


func load_file(filename string, sep string, key_index int) (map[uint64][]*multivalue.MultiValue, error) {
	f, err := os.Open(filename)
	if err != nil { return nil, err }

	data := make(map[uint64][]*multivalue.MultiValue)

	var lineno uint64
	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		lineno += 1
		line := scanner.Text()
		parts := strings.Split(line, sep)

		// make sure the key is in parts
		if len(parts) <= key_index {
			log.Warningf("line %d does not have enough fields", lineno)
			continue
		}

		// convert parts to the right type
		tparts := make([]*multivalue.MultiValue, len(parts))
		for i, part := range parts {
			if global_data_types[i] == CT_STRING {
				tparts[i] = multivalue.NewString(part)

			} else if global_data_types[i] == CT_FLOAT64 {
				v, err := strconv.ParseFloat(part, 64)
				if err != nil {
					log.Warningf("line %d field \"%s\" cannot be parsed as float", lineno, part)
					tparts[i] = global_data_defaults[i]
				} else {
					tparts[i] = multivalue.NewFloat64(v)
				}

			} else if global_data_types[i] == CT_UINT64_10 {
				v, err := strconv.ParseUint(part, 10, 64)
				if err != nil {
					log.Warningf("line %d field \"%s\" cannot be parsed as integer with base 10", lineno, part)
					tparts[i] = global_data_defaults[i]
				} else {
					tparts[i] = multivalue.NewUint64(v)
				}

			} else if global_data_types[i] == CT_UINT64_16 {
				if strings.HasPrefix(part, "0x") { part = part[2:] }
				v, err := strconv.ParseUint(part, 16, 64)
				if err != nil {
					log.Warningf("line %d field \"%s\" cannot be parsed as integer with base 16", lineno, part)
					tparts[i] = global_data_defaults[i]
				} else {
					tparts[i] = multivalue.NewUint64(v)
				}

			} else {
				log.Criticalf("Unknown data type %d", global_data_types[i])
				os.Exit(100)
			}
		}

		// locate the index and make sure it is integer
		mvkey := tparts[key_index]
		if !mvkey.IsUint64() {
			log.Criticalf("Key is not integer")
			os.Exit(100)
		}

		data[mvkey.GetUint64()] = tparts
	}

	return data, nil
}
