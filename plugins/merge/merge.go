package plugin_merge

import (
	"fmt"
	"os"
	"strings"
	"bufio"
	"strconv"
	"fortisession"
	"foset/plugins/common"
	"github.com/juju/loggo"
)

var log loggo.Logger

var global_data_fields   map[uint64][]string
var global_data_columns  []string

func InitPlugin(data string, data_request *fortisession.SessionDataRequest) ( map[string]func(*fortisession.Session)(bool), error) {

	defaults := make(map[string]string)
	defaults["sep"] = " "
	defaults["file"] = ""
	defaults["key"] = "serial"
	dk, du, dui := plugin_common.ExtractData(data, []string{"sep","file","key","force_key_hex"}, defaults)
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

	// find index of serial field
	// at the same time, find the highest column index
	var key_index int  = -1
	var max_index int  = -1

	for k, v := range dui {
		if v == dk["key"] {  key_index = k-1 }
		if k > max_index  {  max_index = k   }
	}

	if key_index == -1 {
		return nil, fmt.Errorf("key field name \"%s\" not present in fields", dk["key"])
	}

	// prepare column names array
	// and sae them to globals
	data_columns := make([]string, max_index)

	for i := 1; i <= max_index; i++ {
		data_columns[i-1], _ = dui[i]
	}

	global_data_columns = data_columns

	// force hex parse even if it does not start with 0x ?
	_, force_key_hex := dk["force_key_hex"]

	// load the data from file and save it to global variable
	data_fields, err := load_file(dk["file"], dk["sep"], key_index, force_key_hex)
	if err != nil { return nil, err }
	global_data_fields = data_fields

	// request fields
	data_request.Custom     = true
	data_request.Serial     = true

	fces := make(map[string]func(*fortisession.Session)(bool))
	fces["beforeFilter"] = ProcessSession

	return fces, nil
}

func ProcessSession(session *fortisession.Session) bool {
	data, found := global_data_fields[session.Serial]

	for i, field := range global_data_columns {
		if len(field) == 0  { continue }

		if !found                 { session.Custom[field] = ""
		} else if i >= len(data)  { session.Custom[field] = ""
		} else                    { session.Custom[field] = data[i] }
	}

	return false
}


func load_file(filename string, sep string, key_index int, force_key_hex bool) (map[uint64][]string, error) {
	f, err := os.Open(filename)
	if err != nil { return nil, err }

	data := make(map[uint64][]string)

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

		// recognize format of the key and parse it
		key_text := strings.ToLower(parts[key_index])
		var key uint64
		if strings.HasPrefix(key_text, "0x") || force_key_hex {
			if strings.HasPrefix(key_text, "0x") { key_text = key_text[2:] }
			key, err = strconv.ParseUint(key_text, 16, 64)
			if err != nil {
				log.Warningf("line %d has unparsable hex session key \"%s\"", lineno, key_text)
				continue
			}
		} else {
			key, err = strconv.ParseUint(key_text, 10, 64)
			if err != nil {
				log.Warningf("line %d has unparsable session key \"%s\"", lineno, key_text)
				continue
			}
		}

		data[key] = parts
	}

	return data, nil
}
