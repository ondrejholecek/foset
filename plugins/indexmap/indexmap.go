// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

// Plugin indexmap can translate the indexes of VDOMs and/or interface to their human readable
// name. 
//
// It accepts parameters "vdoms" and/or "interfaces" that specify the path to the file containg 
// the output of "diag sys vd list" or "diag netlink interface list".
//
// For each session the plugin creates custom value (string based) called "vdom" if "vdoms"
// parameter was given and/or "iface[??]" (check formatter help for the options) if "interfaces"
// parameter was given. If the index wasn't found in the parsed files, the custom variable
// is still present but contain the index as string.
package plugin_indexmap

import (
	"fmt"
	"strconv"
	"bufio"
	"strings"
	"regexp"
	"foset/fortisession"
	"foset/fortisession/multivalue"
	"foset/plugins/common"
	"github.com/juju/loggo"
)

// parameters saved from InitPlugin
var log loggo.Logger

var plugin *plugin_common.FosetPlugin

// dictionaries
var dict_vdoms       map[uint32]string
var dict_interfaces  map[uint32]string

//
func InitPlugin(pluginInfo *plugin_common.FosetPlugin, data string, data_request *fortisession.SessionDataRequest, custom_log loggo.Logger) (error) {
	// setup logging with custom name (to differentiate from other plugins)
	log = custom_log.Child("indexmap")

	// save our plugin info
	plugin = pluginInfo

	// parse data parameters
	defaults := make(map[string]string)
	dk, du, _ := plugin_common.ExtractData(data, []string{"vdoms","interfaces"}, defaults)

	// validate parameters
	unknowns := make([]string, 0)
	for k, _ := range du { unknowns = append(unknowns, k) }
	if len(unknowns) > 0 {
		return fmt.Errorf("following parameters are not recognized: %s", strings.Join(unknowns, ", "))
	}

	// what to do?
	vdoms, _ := dk["vdoms"]
	if vdoms != "" {
		err := parseVdoms(vdoms)
		if err != nil { return fmt.Errorf("cannot parse vdoms file: %s", err) }
		data_request.Policy = true
		data_request.Custom = true
	}

	interfaces, _ := dk["interfaces"]
	if interfaces != "" {
		err := parseInterfaces(interfaces)
		if err != nil { return fmt.Errorf("cannot parse interfaces file: %s", err) }
		data_request.Interfaces = true
		data_request.Custom = true
	}

	// setup callbacks
	var hooks plugin_common.Hooks
	hooks.BeforeFilter = ProcessBeforeFilter

	pluginInfo.Hooks = hooks

	//
	return nil
}

func ProcessBeforeFilter(session *fortisession.Session) bool {
	// do we have some vdom map?
	if dict_vdoms != nil {
		name, exists := dict_vdoms[session.Policy.Vdom]
		if exists {
			session.Custom["vdom"] = multivalue.NewString(name)
		} else {
			session.Custom["vdom"] = multivalue.NewString(fmt.Sprintf("%d", session.Policy.Vdom))
		}
	}

	// do we have some interface map?
	if dict_interfaces != nil {
		var name   string
		var exists bool

		name, exists = dict_interfaces[session.Interfaces.In_org]
		if exists {
			session.Custom["iface[oi]"] = multivalue.NewString(name)
			session.Custom["iface[io]"] = multivalue.NewString(name)
		} else {
			session.Custom["iface[oi]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.In_org))
			session.Custom["iface[io]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.In_org))
		}

		name, exists = dict_interfaces[session.Interfaces.Out_org]
		if exists {
			session.Custom["iface[oo]"] = multivalue.NewString(name)
		} else {
			session.Custom["iface[oo]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.Out_org))
		}

		name, exists = dict_interfaces[session.Interfaces.In_rev]
		if exists {
			session.Custom["iface[ri]"] = multivalue.NewString(name)
			session.Custom["iface[ir]"] = multivalue.NewString(name)
		} else {
			session.Custom["iface[ri]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.In_rev))
			session.Custom["iface[ir]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.In_rev))
		}

		name, exists = dict_interfaces[session.Interfaces.Out_rev]
		if exists {
			session.Custom["iface[ro]"] = multivalue.NewString(name)
			session.Custom["iface[or]"] = multivalue.NewString(name)
		} else {
			session.Custom["iface[ro]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.Out_rev))
			session.Custom["iface[or]"] = multivalue.NewString(fmt.Sprintf("%d", session.Interfaces.Out_rev))
		}

	}

	return false
}

// parsing functions
func parseVdoms(filename string) error {
	f, err := plugin.Inputs.Provide(filename)
	if err != nil { return err }

	re := regexp.MustCompile("^name=([^/]+).*?\\sindex=([0-9]+)")
	dict_vdoms = make(map[uint32]string)

	var inside bool = false
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// the section we are interested in starts with "list virtual ..." line
		if strings.HasPrefix(line, "list virtual firewall info:") { inside = true }
		// ... end ends with an empty line
		if inside && line == "" { return nil }
		// otherwise we are not interested
		if !inside { continue }

		results := re.FindStringSubmatch(line)
		if len(results) == 0 { continue }

		name  := results[1]
		tmp, err := strconv.ParseUint(results[2], 10, 32)
		if err != nil {
			log.Errorf("Cannot parse index \"%s\" for VDOM \"%s\": %s", results[2], name, err)
			continue
		}
		index := uint32(tmp)

		dict_vdoms[index] = name
	}

	return nil
}

func parseInterfaces(filename string) error {
	f, err := plugin.Inputs.Provide(filename)
	if err != nil { return err }

	// if=mgmt1 family=00 type=1 index=3 mtu=1500 link=0 master=0
	re := regexp.MustCompile("^if=([^ ]+)\\s+family=.*?\\sindex=([0-9]+)")
	dict_interfaces = make(map[uint32]string)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		results := re.FindStringSubmatch(line)
		if len(results) == 0 { continue }

		name  := results[1]
		tmp, err := strconv.ParseUint(results[2], 10, 32)
		if err != nil {
			log.Errorf("Cannot parse index \"%s\" for interface \"%s\": %s", results[2], name, err)
			continue
		}
		index := uint32(tmp)

		dict_interfaces[index] = name
	}

	return nil
}
