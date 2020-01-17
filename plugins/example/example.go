// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

// Foset example plugin: intended to show how Foset plugin system works.
//
// This plugin calculates the number of networks used as source IP addresses.
//
// The network is identified by the prefix length which is passed in via `prefixlen` parameter (or
// /24 by default).
//
// The number of networks is calculated for all the sessions as well as for the sessions matching
// the Foset filter. If `hide` parameter is present, this plugin will prevent Foset from display
// the session on its standard output.
//
// The statistics is written after all session are processed.
//
// Usage
//
// Do not show the sessions (only show the statistics at the end) and match networks by CIDR /8:
//    $ foset [...] -p 'example|hide,prefixlen=8' [...]
//
// Show sessions and at the end show also the statistics. Use default CIDR /24:
//    $ foset [...] -p 'example' [...]
//
package plugin_example

import (
	"fmt"
	"strconv"
	"encoding/binary"
	"fortisession"
	"foset/plugins/common"
	"github.com/juju/loggo"
	"net"
	"math"
)

// parameters saved from InitPlugin
var log loggo.Logger

var global_prefixlen uint32
var global_mask      uint32
var global_hide      bool

// parameters used to track progress
var count_total uint64
var networks_before_filter  map[uint32]uint64
var networks_after_filter   map[uint32]uint64

// InitPlugin is the "well-known" entry point of the module that is called from `foset`.
//
// Parameters
//
// `data` Plugin configuration (string after the first `|` in the value of `-p` or `-P` parameters).
// Its format is completely free but if ExtractData function from the common plugin package is used,
// it must follow the key=value structure, delimited by coma(s). See GoDoc for `plugin_common`.
//
// `data_request` Structure specifying which session fields the `foset` must parse before calling the
// plugin. If not specified here, they may not be present in the `Session` structure passed
// into the hook callback functions (but some other module can enable it).
// Plugin must never disable any of its flags - it should only enable new flags. 
//
// `custom_log` Object to use for logging messages. This one should be used instead of definining a private one,
// because the user may enable debugging, tracing, etc. globally.
// 
// Returns
//
// Reference to `FosetPlugin` which specifies to callback functions that should be called at the right time.
// Callback not configured should be kept at `nil`.
//
// Possible error. When not `nil` it will be reported to user and `foset` will terminate.
func InitPlugin(data string, data_request *fortisession.SessionDataRequest, custom_log loggo.Logger) (*plugin_common.FosetPlugin, error) {
	// setup logging with custom name (to differentiate from other plugins)
	log = custom_log.Child("example")

	// parse data parameters
	defaults := make(map[string]string)
	defaults["prefixlen"] = "24"
	dk, du, dui := plugin_common.ExtractData(data, []string{"prefixlen","hide"}, defaults)

	// validate parameters
	if len(du) > 0 || len(dui) > 0 {
		return nil, fmt.Errorf("some unknown parameters received")
	}

	// prefix length
	prefixlen, err := strconv.ParseUint(dk["prefixlen"], 10, 32)
	if err != nil { return nil, fmt.Errorf("parameter prefixlen unparsable: %s", err) }
	global_prefixlen = uint32(prefixlen)

	// calculate the mask
	global_mask = uint32(math.Pow(float64(2), float64(global_prefixlen))) - 1
	global_mask <<= 32-global_prefixlen

	// hide the sessions?
	_, global_hide = dk["hide"]

	// request fields
	data_request.Hooks = true

	// setup callbacks
	var hooks plugin_common.Hooks
	hooks.BeforeFilter = ProcessBeforeFilter
	hooks.AfterFilter  = ProcessAfterFilter
	hooks.Finished     = ProcessFinished

	var pluginInfo plugin_common.FosetPlugin
	pluginInfo.Hooks = hooks

	// initialize globals
	networks_before_filter = make(map[uint32]uint64)
	networks_after_filter  = make(map[uint32]uint64)

	//
	return &pluginInfo, nil
}

// ProcessBeforeFilter is called immediatelly after the `Session` is parsed.
func ProcessBeforeFilter(session *fortisession.Session) bool {
	// here we can count total session loaded from file
	count_total++

	// and those matching the prefix
	src_ip, _, _, _, _, _, _ := session.GetPeers()
	net := getNetwork(src_ip)

	networks_before_filter[net] += 1

	// even if we want to hide sessions from output, we cannot return `true`
	// from `BeforeFilter` callback, because then the `AfterFilter` would
	// never be called!
	return false
}

// ProcessAfterFilter is called when (if) the `Session` matches the user
// specified filter.
func ProcessAfterFilter(session *fortisession.Session) bool {
	src_ip, _, _, _, _, _, _ := session.GetPeers()
	net := getNetwork(src_ip)

	networks_after_filter[net] += 1

	// here we can decide whether the session should be shown on terminal
	return global_hide
}

// ProcessFinished is called when all the sessions are processed
// and `foset` is about to terminate.
func ProcessFinished() {
	fmt.Printf("Statistics about source IP addresses:\n")
	fmt.Printf("\tTotal sessions                  : %d\n", count_total)
	fmt.Printf("\tNetwork prefix length           : %d\n", global_prefixlen)
	fmt.Printf("\tTotal unique networks           : %d\n", len(networks_before_filter))
	fmt.Printf("\tUnique networks matching filter : %d\n", len(networks_after_filter))
}


// Local functions
func getNetwork(ip net.IP) uint32 {
	ip  = ip.To4()
	num := binary.BigEndian.Uint32(ip)
	num &= global_mask
	return num
}
