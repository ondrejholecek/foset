// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package plugin_stats

import (
	"os"
	"fmt"
	"strings"
	"strconv"
	"math"
	"net"
	"path"
	"time"
	"encoding/binary"
	"sync/atomic"
	"fortisession"
	"foset/plugins/common"
	"github.com/juju/loggo"
)

// parameters saved from InitPlugin
var log loggo.Logger
var plugin_filename string
var plugin_filter   string
var plugin_version  string
var plugin_commit   string
var config          string

// plugin parameters
var directory            string
var directory_override   bool
var use_complex_matching bool
var srcprefix, dstprefix uint32
var srcmask, dstmask     uint32
var translate_vdoms      bool
var translate_interfaces bool

// counters
var tcpsrcports, tcpdstports, tcpsrcdstports *Counter
var udpsrcports, udpdstports, udpsrcdstports *Counter

var srcnetworks, dstnetworks, srcdstnetworks *Counter
var srcnetworks_rate, dstnetworks_rate, srcdstnetworks_rate *Counter
var srcnetworks_bytes, dstnetworks_bytes, srcdstnetworks_bytes *Counter
var srcnetworks_counts, dstnetworks_counts, srcdstnetworks_counts *Counter
var srcnetworks_errs, dstnetworks_errs, srcdstnetworks_errs *Counter
var snat_ip, snat_port, snat_ipport *Counter

var protocols *Counter
var vdoms *Counter
var policies *Counter
var durations *Counter
var ttls *Counter
var helpers *Counter
var users *Counter
var states *Counter
var tcp_sstateL, tcp_sstateR, tcp_sstateLR *Counter
var udp_sstateL, udp_sstateR, udp_sstateLR *Counter

var shapers_org, shapers_rev, shapers_perip *Counter

var tunnels_in, tunnels_out *Counter
var interfaces_org_in, interfaces_org_out, interfaces_rev_in, interfaces_rev_out *Counter
var nexthop_org, nexthop_rev *Counter

var offload_npu, offload_nturbo *Counter
var offload_fail, offload_fail_org, offload_fail_rev *Counter

// dictionaries definition
var dict_tcp_session_state map[uint8]string
var dict_udp_session_state map[uint8]string

// session counters
var count_total    uint64  // all sessions
var count_matched  uint64  // session matching filter

//
func InitPlugin(data string, data_request *fortisession.SessionDataRequest, custom_log loggo.Logger) (*plugin_common.FosetPlugin, error) {
	// setup logging with custom name (to differentiate from other plugins)
	log = custom_log.Child("stats")

	translate_vdoms = true
	translate_interfaces = true

	// parse data parameters
	defaults := make(map[string]string)
	defaults["srcprefix"] = "24"
	defaults["dstprefix"] = "24"
	dk, du, _ := plugin_common.ExtractData(data, []string{"srcprefix","dstprefix","complex","directory","force","transvdoms","transifaces"}, defaults)

	// validate parameters
	unknowns := make([]string, 0)
	for k, _ := range du { unknowns = append(unknowns, k) }
	if len(unknowns) > 0 {
		return nil, fmt.Errorf("following parameters are not recognized: %s", strings.Join(unknowns, ", "))
	}

	// translations
	_, translate_vdoms      = dk["transvdoms"]
	_, translate_interfaces = dk["transifaces"]

	var err error
	var exists bool

	// save the path to the results directory
	directory, exists = dk["directory"]
	if !exists {
		return nil, fmt.Errorf("parameter \"directory\" is mandatory")
	}

	// if the directory does not exist, we create it (_override = true)
	// if the path exists and it is not a directory we return error
	// otherwise the path exists and is a directory, so we don't override it
	dstat, _ := os.Stat(directory)
	if dstat == nil {
		directory_override = true
	} else if !dstat.IsDir() {
		return nil, fmt.Errorf("directory path \"%s\" is not a directory", directory)
	}

	// however, it the "force" parameter was specified, override it it anyway
	_, exists = dk["force"]
	if exists {
		log.Debugf("Parameter \"force\" present, overriding/recreating directory \"%s\"", directory)
		directory_override = true
	}

	// prefix length specifies how precies should be the network calculations
	var tmp uint64

	tmp, err = strconv.ParseUint(dk["srcprefix"], 10, 32)
	if err != nil { return nil, fmt.Errorf("parameter srcprefix unparsable: %s", err) }
	if tmp == 0 || tmp > 32 { return nil, fmt.Errorf("nonsence srcprefix length %d", tmp) }
	srcprefix = uint32(tmp)
	srcmask   = uint32(math.Pow(float64(2), float64(srcprefix))-1) << (32-srcprefix)

	tmp, err = strconv.ParseUint(dk["dstprefix"], 10, 32)
	if err != nil { return nil, fmt.Errorf("parameter dstprefix unparsable: %s", err) }
	if tmp == 0 || tmp > 32 { return nil, fmt.Errorf("nonsence dstprefix length %d", tmp) }
	dstprefix = uint32(tmp)
	dstmask   = uint32(math.Pow(float64(2), float64(dstprefix))-1) << (32-dstprefix)

	// by default the complex matching (networks-to-networks, ports-to-ports) are disabled
	// because those are extremely memory exhausing
	_, use_complex_matching = dk["complex"]

	// save the public parts of config string
	config = fmt.Sprintf("srcprefix=%d,dstprefix=%d", srcprefix, dstprefix)
	if use_complex_matching { config += ",complex" }
	if translate_vdoms      { config += ",transvdoms" }
	if translate_interfaces { config += ",transifaces" }

	// request fields
	data_request.Hooks      = true
	data_request.Basics     = true
	data_request.Rate       = true
	data_request.Stats      = true
	data_request.Policy     = true
	data_request.Interfaces = true
	data_request.Npu        = true
	data_request.NpuError   = true
	data_request.Other      = true
	data_request.Auth       = true
	data_request.States     = true
	data_request.Shaping    = true

	// setup callbacks
	var hooks plugin_common.Hooks
	hooks.Start        = Start
	hooks.BeforeFilter = ProcessBeforeFilter
	hooks.AfterFilter  = ProcessAfterFilter
	hooks.Finished     = ProcessFinished

	var pluginInfo plugin_common.FosetPlugin
	pluginInfo.Hooks = hooks

	// initialize counters
	tcpsrcports           = CounterInit("tcp_src_ports", WriteSimpleData)
	tcpdstports           = CounterInit("tcp_dst_ports", WriteSimpleData)
	tcpsrcdstports        = CounterInit("tcp_srcdst_ports", WriteSimpleData)
	udpsrcports           = CounterInit("udp_src_ports", WriteSimpleData)
	udpdstports           = CounterInit("udp_dst_ports", WriteSimpleData)
	udpsrcdstports        = CounterInit("udp_srcdst_ports", WriteSimpleData)
	protocols             = CounterInit("protocols", WriteSimpleData)
	vdoms                 = CounterInit("vdoms", WriteSimpleData)
	policies              = CounterInit("policies", WriteSimpleData)
	srcnetworks           = CounterInit("src_net", WriteSimpleData)
	dstnetworks           = CounterInit("dst_net", WriteSimpleData)
	srcdstnetworks        = CounterInit("srcdst_net", WriteSimpleData)
	srcnetworks_rate      = CounterInit("src_net_rate", WriteSimpleData)
	dstnetworks_rate      = CounterInit("dst_net_rate", WriteSimpleData)
	srcdstnetworks_rate   = CounterInit("srcdst_net_rate", WriteSimpleData)
	srcnetworks_bytes     = CounterInit("src_net_bytes", WriteSimpleData)
	dstnetworks_bytes     = CounterInit("dst_net_bytes", WriteSimpleData)
	srcdstnetworks_bytes  = CounterInit("srcdst_net_bytes", WriteSimpleData)
	srcnetworks_counts    = CounterInit("src_net_counts", WriteSimpleData)
	dstnetworks_counts    = CounterInit("dst_net_counts", WriteSimpleData)
	srcdstnetworks_counts = CounterInit("srcdst_net_counts", WriteSimpleData)
	srcnetworks_errs      = CounterInit("src_net_errs", WriteSimpleData)
	dstnetworks_errs      = CounterInit("dst_net_errs", WriteSimpleData)
	srcdstnetworks_errs   = CounterInit("srcdst_net_errs", WriteSimpleData)
	nexthop_org           = CounterInit("nexthop_org", WriteSimpleData)
	nexthop_rev           = CounterInit("nexthop_rev", WriteSimpleData)
	durations             = CounterInit("duration", WriteSimpleData)
	ttls                  = CounterInit("ttl", WriteSimpleData)
	helpers               = CounterInit("helper", WriteSimpleData)
	users                 = CounterInit("user", WriteSimpleData)
	states                = CounterInit("state", WriteSimpleData)
	offload_npu           = CounterInit("offload_npu", WriteSimpleData)
	offload_nturbo        = CounterInit("offload_nturbo", WriteSimpleData)
	offload_fail          = CounterInit("offload_fail", WriteSimpleData)
	offload_fail_org      = CounterInit("offload_fail_org", WriteSimpleData)
	offload_fail_rev      = CounterInit("offload_fail_rev", WriteSimpleData)
	tunnels_in            = CounterInit("tunnel_in", WriteSimpleData)
	tunnels_out           = CounterInit("tunnel_out", WriteSimpleData)
	interfaces_org_in     = CounterInit("interface_org_in", WriteSimpleData)
	interfaces_org_out    = CounterInit("interface_org_out", WriteSimpleData)
	interfaces_rev_in     = CounterInit("interface_rev_in", WriteSimpleData)
	interfaces_rev_out    = CounterInit("interface_rev_out", WriteSimpleData)
	shapers_org           = CounterInit("shaper_org", WriteSimpleData)
	shapers_rev           = CounterInit("shaper_rev", WriteSimpleData)
	shapers_perip         = CounterInit("shaper_perip", WriteSimpleData)
	snat_ip               = CounterInit("snat_ip", WriteSimpleData)
	snat_port             = CounterInit("snat_port", WriteSimpleData)
	snat_ipport           = CounterInit("snat_ipport", WriteSimpleData)
	tcp_sstateL           = CounterInit("tcp_sstate_l", WriteSimpleData)
	tcp_sstateR           = CounterInit("tcp_sstate_r", WriteSimpleData)
	tcp_sstateLR          = CounterInit("tcp_sstate_lr", WriteSimpleData)
	udp_sstateL           = CounterInit("udp_sstate_l", WriteSimpleData)
	udp_sstateR           = CounterInit("udp_sstate_r", WriteSimpleData)
	udp_sstateLR          = CounterInit("udp_sstate_lr", WriteSimpleData)

	// initialize and fill maps with dictionaries
	initDictTCPState();
	initDictUDPState();

	//
	return &pluginInfo, nil
}

func Start(pi *plugin_common.FosetPlugin) {
	plugin_filename = path.Base(pi.Filename)
	plugin_filter   = pi.Filter
	plugin_version  = pi.Version
	plugin_commit   = pi.Commit
}

func ProcessBeforeFilter(session *fortisession.Session) bool {
	atomic.AddUint64(&count_total, 1)
	return false
}

func ProcessAfterFilter(session *fortisession.Session) bool {
	atomic.AddUint64(&count_matched, 1)

	// It is expected that all sessions either have the custom vdom and iface[??] parameters
	// or none of them have it. (For indexes that were not found, it is still expected to be
	// present and contain the index in the text format.)
	// Hence, we check whether the custom is field is present and if not, we disable translation.
	if translate_vdoms {
		_, custom_vdom := session.Custom["vdom"]
		if !custom_vdom {
			log.Errorf("VDOMs translation enabled, but at least one session does not have the custom field - disabling feature")
			translate_vdoms = false
		}

		_, custom_iface := session.Custom["iface[oo]"]
		if !custom_iface {
			log.Errorf("Interface translation enabled, but at least one session does not have the custom field - disabling feature")
			translate_interfaces = false
		}
	}

	//
	src_ip, src_port, dst_ip, dst_port, nat_ip, nat_port, _ := session.GetPeers()
	srcnet := getNetwork(src_ip, srcmask)
	dstnet := getNetwork(dst_ip, dstmask)

	snatip := getNetwork(nat_ip, 0xffffffff)
	snat_ip.AddOne(snatip)
	snat_port.AddOne(nat_port)

	protocols.AddOne(session.Basics.Protocol)

	if translate_vdoms {
		vdoms.AddOne(session.Custom["vdom"].AsString())
		if session.Policy.Id == 0xffffffff {
			policies.AddOne(fmt.Sprintf("%s / (i)", session.Custom["vdom"].AsString()))
		} else {
			policies.AddOne(fmt.Sprintf("%s / %d", session.Custom["vdom"].AsString(), session.Policy.Id))
		}
	} else {
		vdoms.AddOne(session.Policy.Vdom)
		if session.Policy.Id == 0xffffffff {
			policies.AddOne(fmt.Sprintf("%d / (i)", session.Policy.Vdom))
		} else {
			policies.AddOne(fmt.Sprintf("%d / %d", session.Policy.Vdom, session.Policy.Id))
		}
	}

	if session.Basics.Protocol == 6 {
		tcpsrcports.AddOne(src_port)
		tcpdstports.AddOne(dst_port)

		if use_complex_matching {
			tcpsrcdstports.AddOne(uint64(src_port) << 16 | uint64(dst_port))
		}

		tcp_sstateL.AddOne(session.Basics.StateL)
		tcp_sstateR.AddOne(session.Basics.StateR)
		tcp_sstateLR.AddOne(uint16(session.Basics.StateL) << 8 | uint16(session.Basics.StateR))

	} else if session.Basics.Protocol == 17 {
		udpsrcports.AddOne(src_port)
		udpdstports.AddOne(dst_port)

		if use_complex_matching {
			udpsrcdstports.AddOne(uint64(src_port) << 16 | uint64(dst_port))
		}

		udp_sstateL.AddOne(session.Basics.StateL)
		udp_sstateR.AddOne(session.Basics.StateR)
		udp_sstateLR.AddOne(uint16(session.Basics.StateL) << 8 | uint16(session.Basics.StateR))
	}

	srcnetworks.AddOne(srcnet)
	dstnetworks.AddOne(dstnet)
	srcnetworks_rate.Add(srcnet, session.Rate.Tx_Bps)
	dstnetworks_rate.Add(dstnet, session.Rate.Rx_Bps)
	srcnetworks_bytes.Add(srcnet, session.Stats.Bytes_org)
	dstnetworks_bytes.Add(dstnet, session.Stats.Bytes_rev)
	srcnetworks_counts.Add(srcnet, session.Stats.Packets_org)
	dstnetworks_counts.Add(dstnet, session.Stats.Packets_rev)
	srcnetworks_errs.Add(srcnet, session.Stats.Errors_org)
	dstnetworks_errs.Add(dstnet, session.Stats.Errors_rev)
	nexthop_org.AddOne(getNetwork(session.Interfaces.NextHop_org, 0xffffffff))
	nexthop_rev.AddOne(getNetwork(session.Interfaces.NextHop_rev, 0xffffffff))

	if use_complex_matching {
		srcdstnet  := uint64(srcnet) << 32 | uint64(dstnet)

		srcdstnetworks.AddOne(srcdstnet)
		srcdstnetworks_rate.Add(srcdstnet, session.Rate.Tx_Bps + session.Rate.Rx_Bps)
		srcdstnetworks_bytes.Add(srcdstnet, session.Stats.Bytes_org + session.Stats.Bytes_rev)
		srcdstnetworks_counts.Add(srcdstnet, session.Stats.Packets_org + session.Stats.Packets_rev)
		srcdstnetworks_errs.Add(srcdstnet, session.Stats.Errors_org + session.Stats.Errors_rev)

		snatipport := uint64(snatip) << 32 | uint64(nat_port)
		snat_ipport.AddOne(snatipport)
	}

	switch duration := session.Basics.Duration; {
		case duration <= 10:
			durations.AddOne(uint64(10))
		case duration <= 60:
			durations.AddOne(uint64(60))
		case duration <= 300:
			durations.AddOne(uint64(300))
		case duration <= 900:
			durations.AddOne(uint64(900))
		case duration <= 3600:
			durations.AddOne(uint64(3600))
		case duration <= 3*3600:
			durations.AddOne(uint64(3*3600))
		case duration <= 6*3600:
			durations.AddOne(uint64(6*3600))
		case duration <= 12*3600:
			durations.AddOne(uint64(12*3600))
		case duration <= 24*3600:
			durations.AddOne(uint64(24*3600))
		case duration <= 48*3600:
			durations.AddOne(uint64(48*3600))
		case duration <= 7*24*3600:
			durations.AddOne(uint64(7*24*3600))
		default:
			durations.AddOne(uint64(0xffffffffffffffff))
	}

	for _, s := range session.States {
		states.AddOne(string(s))
	}

	ttls.AddOne(session.Basics.Timeout)
	helpers.AddOne(session.Other.Helper)
	users.AddOne(session.Auth.User)

	var offload_npu_mix uint64 = (uint64(session.Npu.Offload_org) << 8) | uint64(session.Npu.Offload_rev)
	offload_npu.AddOne(offload_npu_mix)
	var offload_nturbo_mix uint64 = (uint64(session.Npu.Nturbo_org) << 8) | uint64(session.Npu.Nturbo_rev)
	offload_nturbo.AddOne(offload_nturbo_mix)

	if session.NpuError.NoOffloadReason != "" {
		offload_fail.AddOne(session.NpuError.NoOffloadReason)
	}

	if (session.NpuError.Kernel_org != "" && session.NpuError.Driver_org != "") {
		offload_fail_org.AddOne(session.NpuError.Kernel_org + "/" + session.NpuError.Driver_org)
	}

	if (session.NpuError.Kernel_rev != "" && session.NpuError.Driver_rev != "") {
		offload_fail_rev.AddOne(session.NpuError.Kernel_rev + "/" + session.NpuError.Driver_rev)
	}

	if session.Other.Tunnel_in != "" {
		tunnels_in.AddOne(session.Other.Tunnel_in)
	}

	if session.Other.Tunnel_out != "" {
		tunnels_out.AddOne(session.Other.Tunnel_out)
	}

	if translate_interfaces {
		interfaces_org_in.AddOne(session.Custom["iface[oi]"].AsString())
		interfaces_org_out.AddOne(session.Custom["iface[oo]"].AsString())
		interfaces_rev_in.AddOne(session.Custom["iface[ri]"].AsString())
		interfaces_rev_out.AddOne(session.Custom["iface[ro]"].AsString())
	} else {
		interfaces_org_in.AddOne(session.Interfaces.In_org)
		interfaces_org_out.AddOne(session.Interfaces.Out_org)
		interfaces_rev_in.AddOne(session.Interfaces.In_rev)
		interfaces_rev_out.AddOne(session.Interfaces.Out_rev)
	}

	shapers_org.AddOne(session.Shaping.Shaper_org)
	shapers_rev.AddOne(session.Shaping.Shaper_rev)
	shapers_perip.AddOne(session.Shaping.Shaper_ip)

	return true
}

// ProcessFinished is called when all the sessions are processed
// and `foset` is about to terminate.
func ProcessFinished() {
	// if we are supposed to create/override the directory, do that
	if directory_override {
		log.Debugf("Overriding (or creating) directory \"%s\"", directory)
		createDirectory(directory)
	}
	// open the JS data file inside the directory
	f, err := os.OpenFile(path.Join(directory, "resources/data.js"), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Criticalf("unable to write graph data: %s", err)
		return
	}

	// Setup the JavaScript object.
	// Because data is appened to the .js file, during each run we first create `current` object 
	// and save all data to it. At the end we save it to the `foset` object using unique name.
	fmt.Fprintf(f, "var current = Object()\n")
	fmt.Fprintf(f, "current.info = Object()\n")
	fmt.Fprintf(f, "current.info.filename         = \"%s\"\n", plugin_filename)
	fmt.Fprintf(f, "current.info.filter           = \"%s\"\n", plugin_filter)
	fmt.Fprintf(f, "current.info.version          = \"%s\"\n", plugin_version)
	fmt.Fprintf(f, "current.info.commit           = \"%s\"\n", plugin_commit)
	fmt.Fprintf(f, "current.info.plugin_config    = \"%s\"\n", config)
	fmt.Fprintf(f, "current.info.calculated       = %d\n", time.Now().Unix())
	fmt.Fprintf(f, "current.info.sessions_total   = %d\n", count_total)
	fmt.Fprintf(f, "current.info.sessions_matched = %d\n", count_matched)
	fmt.Fprintf(f, "current.data = Object()\n")
	fmt.Fprintf(f, "current.order = []\n")
	fmt.Fprintf(f, "current.tabs  = []\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"general-overview\", \"title\":\"General overview\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"ports\", \"title\":\"Ports\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"networks\", \"title\":\"Networks\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"session-states\", \"title\":\"Session states\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"snat\", \"title\":\"Source NAT\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"data-rates\", \"title\":\"Data rates\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"packet-counts-and-bytes\", \"title\":\"Packet counts &amp; bytes\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"times\", \"title\":\"Times\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"tunnels\", \"title\":\"Tunnels\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"interfaces\", \"title\":\"Interfaces\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"shapers\", \"title\":\"Shapers\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"next-hops\", \"title\":\"Next hops\"});\n")
	fmt.Fprintf(f, "current.tabs.push({\"id\":\"offload\", \"title\":\"Offload\"});\n")

	// Params is a map that is reused many times when we need to dump counter key/value
	// to the JavaScript file. For different counters we might want to specify different
	// parameters, but a lot of them stays (at least for few neighbouring counters).

	params := make(map[string]interface{})

	// Transformations are used to convert the `key` object from the counter
	// to some human readable string.
	// Sometimes it is as easy as converting number to string, but
	// other times the `key` has more complicated behavior
	// (like containg IP address (32 bits) as well as port number (16 bits)).
	transform_ip := func(o interface{})(string) {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, o.(uint32))
		return fmt.Sprintf("%s", ip)
	}

	transform_ipport := func(o interface{})(string) {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, uint32(o.(uint64) >> 32))
		return fmt.Sprintf("%s : %d", ip, uint8(o.(uint64)))
	}

	transform_srcnet := func(o interface{})(string) {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, o.(uint32))
		return fmt.Sprintf("%s/%d", ip, srcprefix)
	}

	transform_dstnet := func(o interface{})(string) {
		ip := make(net.IP, 4)
		binary.BigEndian.PutUint32(ip, o.(uint32))
		return fmt.Sprintf("%s/%d", ip, dstprefix)
	}

	transform_srcdstnet := func(o interface{})(string) {
		src := uint32(o.(uint64) >> 32)
		dst := uint32(o.(uint64))
		srcip := make(net.IP, 4)
		dstip := make(net.IP, 4)
		binary.BigEndian.PutUint32(srcip, src)
		binary.BigEndian.PutUint32(dstip, dst)
		return fmt.Sprintf("%s/%d <-> %s/%d", srcip, srcprefix, dstip, dstprefix)
	}

	transform_srcdstport := func(o interface{})(string) {
		src_port := uint16(o.(uint64) >> 16)
		dst_port := uint16(o.(uint64))
		return fmt.Sprintf("%d -> %d", src_port, dst_port)
	}

	transform_tcp_session_state := func(o interface{})(string) {
		name, exists := dict_tcp_session_state[o.(uint8)]
		if !exists { return "UNKNOWN"
		} else { return name }
	}

	transform_tcp_combined_session_state := func(o interface{})(string) {
		left  := uint8(o.(uint16) >> 8)
		right := uint8(o.(uint16))
		lname, lexists := dict_tcp_session_state[left]
		rname, rexists := dict_tcp_session_state[right]
		var r string

		if !lexists { r += "UNKNOWN"
		} else { r += lname }
		r += " / "
		if !rexists { r += "UNKNOWN"
		} else { r += rname }

		return r
	}

	transform_udp_combined_session_state := func(o interface{})(string) {
		left  := uint8(o.(uint16) >> 8)
		right := uint8(o.(uint16))
		lname, lexists := dict_udp_session_state[left]
		rname, rexists := dict_udp_session_state[right]
		var r string

		if !lexists { r += "UNKNOWN"
		} else { r += lname }
		r += " / "
		if !rexists { r += "UNKNOWN"
		} else { r += rname }

		return r
	}

	transform_udp_session_state := func(o interface{})(string) {
		name, exists := dict_udp_session_state[o.(uint8)]
		if !exists { return "UNKNOWN"
		} else { return name }
	}


	transform_lifetime := func(o interface{})(string) {
		switch duration := o.(uint64); {
			case duration <= 10:
				return "Less than 10 seconds"
			case duration <= 60:
				return "Between 10 and 60 seconds"
			case duration <= 300:
				return "Between 1 and 5 minutes"
			case duration <= 900:
				return "Between 5 and 15 minutes"
			case duration <= 3600:
				return "Between 15 minutes and 1 hour"
			case duration <= 3*3600:
				return "Between 1 hour and 3 hours"
			case duration <= 6*3600:
				return "Between 3 hours and 6 hours"
			case duration <= 12*3600:
				return "Between 6 hours and 12 hours"
			case duration <= 24*3600:
				return "Between 12 hours and 24 hours"
			case duration <= 48*3600:
				return "Between 24 hours and 48 hours"
			case duration <= 7*24*3600:
				return "Between 2 and 7 days"
			default:
				return "More than 7 days"
		}
	}

	transform_offload := func(o interface{})(string) {
		org := uint8(o.(uint64) >> 8)
		rev := uint8(o.(uint64))
		if org > 0 && rev > 0 { return "Both directions"
		} else if org > 0 && rev == 0 { return "Only original direction"
		} else if org == 0 && rev > 0 { return "Only reverse direction"
		} else { return "No direction" }
	}

	transform_text := func(o interface{})(string) {
		text := o.(string)
		if text == "" { return "[none]" }
		return text
	}


	// Save the counters to the JavaScript file.

	// Top X could be given as parameter of the plugin, but it is rather hardcoded, because the ChartJS
	// library is not very good in plotting variables number of lines into a fixed width canvas :-/
	// Something around 14 is proved to give good results.
	params["top"] = 14

	// General overview
	params["tab"] = "general-overview"

	trans := make(map[string]string)
	trans["1"] = "ICMP"
	trans["6"] = "TCP"
	trans["17"] = "UDP"
	trans["41"] = "IPv6"
	trans["47"] = "GRE"
	trans["50"] = "ESP"
	params["title"] = "Protocols"
	params["translate"] = trans
	params["transform"] = nil
	params["valueformat"] = "number"
	params["showOthers"] = true
	params["description"] = "Number of sessions using the specific IP protocol. Some well-known protocol numbers are automatically translated to their names, the unknown are left as protocol numbers found in IP packet header field 'Protocol'."
	protocols.WriteData(f, params)

	//
	if translate_vdoms {
		params["title"] = "VDOMs"
		params["translate"] = nil
		params["valueformat"] = "number"
		params["showOthers"] = true
		params["transform"] = transform_text
		params["description"] = "Number of sessions per VDOM. VDOM indexes are translated based on provided 'diagnose sys vd list' output."
		vdoms.WriteData(f, params)
	} else {
		params["title"] = "VDOMs"
		params["translate"] = nil
		params["valueformat"] = "number"
		params["showOthers"] = true
		params["transform"] = nil
		params["description"] = "Number of sessions per VDOM. VDOM are represented by number that can be manually translated using 'diagnose sys vd list' FortiGate command (look for 'name' and 'index' fields)."
		vdoms.WriteData(f, params)
	}

	if translate_vdoms {
		params["title"] = "Policies"
		params["transform"] = transform_text
		params["valueformat"] = "number"
		params["showOthers"] = true
		params["description"] = "Number of sessions per policy. Because the policy IDs can be duplicated in different VDOMs, each line is composed of the VDOM name / policy ID. Policy '(i)' means the internal policy that is used for traffic local to FortiGate."
		policies.WriteData(f, params)
	} else {
		params["title"] = "Policies"
		params["transform"] = transform_text
		params["valueformat"] = "number"
		params["showOthers"] = true
		params["description"] = "Number of sessions per policy. Because the policy IDs can be duplicated in different VDOMs, each line is composed of the VDOM index / policy ID. Policy '(i)' means the internal policy that is used for traffic local to FortiGate."
		policies.WriteData(f, params)
	}

	params["title"] = "State flags"
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["showOthers"] = true
	params["description"] = "State flags from each session. If there are more state flags in one session, they are counted independently, hence the total would not match the number of sessions."
	states.WriteData(f, params)

	params["title"] = "Helpers"
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["showOthers"] = true
	params["description"] = "Used helpers on sessions."
	helpers.WriteData(f, params)

	params["title"] = "Users"
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["showOthers"] = true
	params["description"] = "Sessions with authorized users."
	users.WriteData(f, params)

	// Top ports
	params["tab"] = "ports"

	params["title"] = "TCP source ports"
	params["description"] = "Source TCP ports used by the clients."
	params["transform"] = nil
	params["valueformat"] = "number"
	tcpsrcports.WriteData(f, params)

	params["title"] = "TCP destination ports"
	params["description"] = "Destination TCP ports used by the servers."
	params["transform"] = nil
	params["valueformat"] = "number"
	tcpdstports.WriteData(f, params)

	if use_complex_matching {
		params["description"] = "Combinations of source and destionation TCP ports. This might be useful to locate sessions using static ports configuration."
		params["title"] = "TCP source + destination ports"
		params["transform"] = transform_srcdstport
		params["valueformat"] = "number"
		tcpsrcdstports.WriteData(f, params)
	} else {
		WriteSpace(f, "ports")
	}

	params["title"] = "UDP source ports"
	params["description"] = "Source UDP ports used by the clients."
	params["transform"] = nil
	params["valueformat"] = "number"
	udpsrcports.WriteData(f, params)

	params["title"] = "UDP destination ports"
	params["description"] = "Destination UDP ports used by the clients."
	params["transform"] = nil
	params["valueformat"] = "number"
	udpdstports.WriteData(f, params)

	if use_complex_matching {
		params["title"] = "UDP source + destination ports"
		params["description"] = "Combinations of source and destionation UDP ports. This might be useful to locate sessions using static ports configuration."
		params["transform"] = transform_srcdstport
		params["valueformat"] = "number"
		udpsrcdstports.WriteData(f, params)
	} else {
		WriteSpace(f, "ports")
	}

	// Top nets
	params["tab"] = "networks"

	params["title"] = "Source networks"
	params["description"] = fmt.Sprintf("Sessions originating from clients. Grouped by the network prefix specified in the plugin configuration (/%d in this case).", srcprefix)
	params["transform"] = transform_srcnet
	params["valueformat"] = "number"
	srcnetworks.WriteData(f, params)

	params["title"] = "Destination networks"
	params["description"] = fmt.Sprintf("Sessions connected to servers. Grouped by the network prefix specified in the plugin configuration (/%d in this case).", dstprefix)
	params["transform"] = transform_dstnet
	params["valueformat"] = "number"
	dstnetworks.WriteData(f, params)

	if use_complex_matching {
		params["title"] = "Source + destination networks"
		params["description"] = fmt.Sprintf("Sessions between specific clients and servers. Grouped by the network prefix specified in the plugin configuration (/%d for source and /%d for destination IP in this case).", srcprefix, dstprefix)
		params["transform"] = transform_srcdstnet
		params["valueformat"] = "number"
		srcdstnetworks.WriteData(f, params)
	} else {
		WriteSpace(f, "networks")
	}

	params["tab"] = "data-rates"

	params["title"] = "Upload rate from source networks"
	params["description"] = fmt.Sprintf("Total rate of all clients uploading the data. Grouped by the source network prefix specified in plugin configuration (/%d in this case). Be aware that hardware accelerated sessions may not be counted correctly.", srcprefix)
	params["transform"] = transform_srcnet
	params["valueformat"] = "rate"
	srcnetworks_rate.WriteData(f, params)

	params["title"] = "Download rate from destination networks"
	params["description"] = fmt.Sprintf("Total rate of all servers sending the data (clients downloading). Grouped by the destination network prefix specified in plugin configuration (/%d in this case). Be aware that hardware accelerated sessions may not be counted correctly.", dstprefix)
	params["transform"] = transform_dstnet
	params["valueformat"] = "rate"
	dstnetworks_rate.WriteData(f, params)

	if use_complex_matching {
		params["transform"] = transform_srcdstnet
		params["valueformat"] = "rate"
		params["title"] = "Summary rate between source+destination networks"
		params["description"] = fmt.Sprintf("Summarized rate of upload and download traffic between clients and servers.. Grouped by the network prefix specified in the plugin configuration (/%d for source and /%d for destination IP in this case). Be aware that hardware accelerated sessions may not be counted correctly.", srcprefix, dstprefix)
		srcdstnetworks_rate.WriteData(f, params)
	} else {
		WriteSpace(f, "data-rates")
	}

	params["tab"] = "packet-counts-and-bytes"

	params["title"] = "Bytes sent from source networks"
	params["description"] = "Number of bytes already sent by clients in the established sessions. Be careful because the already closed sessions are not calculated. Be aware that hardware accelerated sessions may not be counted correctly."
	params["transform"] = transform_srcnet
	params["valueformat"] = "size"
	srcnetworks_bytes.WriteData(f, params)

	params["title"] = "Bytes received from destination networks"
	params["description"] = "Number of bytes already sent by servers (received by clients) in the established sessions. Be careful because the already closed sessions are not calculated. Be aware that hardware accelerated sessions may not be counted correctly."
	params["transform"] = transform_dstnet
	params["valueformat"] = "size"
	dstnetworks_bytes.WriteData(f, params)

	if use_complex_matching {
		params["transform"] = transform_srcdstnet
		params["valueformat"] = "size"
		params["title"] = "Bytes exchanged between source+destination networks"
		params["description"] = "Sum of bytes already exchanged between clients and servers in the established sessions. Be careful because the already closed sessions are not calculated. Be aware that hardware accelerated sessions may not be counted correctly."
		srcdstnetworks_bytes.WriteData(f, params)
	} else {
		WriteSpace(f, "packet-counts-and-bytes")
	}

	params["title"] = "Packets sent from source networks"
	params["description"] = "Number of packets already sent by clients in the established sessions. Be careful because the already closed sessions are not calculated. Be aware that hardware accelerated sessions may not be counted correctly."
	params["transform"] = transform_srcnet
	params["valueformat"] = "number"
	srcnetworks_counts.WriteData(f, params)

	params["title"] = "Packets received from destination networks"
	params["description"] = "Number of packets already sent by servers (received by clients) in the established sessions. Be careful because the already closed sessions are not calculated. Be aware that hardware accelerated sessions may not be counted correctly."
	params["transform"] = transform_dstnet
	params["valueformat"] = "number"
	dstnetworks_counts.WriteData(f, params)

	if use_complex_matching {
		params["transform"] = transform_srcdstnet
		params["valueformat"] = "number"
		params["title"] = "Packets exchanged between source+destination networks"
		params["description"] = "Sum of packets already exchanged between clients and servers in the established sessions. Be careful because the already closed sessions are not calculated. Be aware that hardware accelerated sessions may not be counted correctly."
		srcdstnetworks_counts.WriteData(f, params)
	} else {
		WriteSpace(f, "packet-counts-and-bytes")
	}

	params["title"] = "Errors from source networks"
	params["description"] = ""
	params["transform"] = transform_srcnet
	params["valueformat"] = "number"
	srcnetworks_errs.WriteData(f, params)

	params["title"] = "Errors from destination networks"
	params["description"] = ""
	params["transform"] = transform_dstnet
	params["valueformat"] = "number"
	dstnetworks_errs.WriteData(f, params)

	if use_complex_matching {
		params["transform"] = transform_srcdstnet
		params["valueformat"] = "number"
		params["title"] = "Errors summary between source+destination networks"
		params["description"] = ""
		srcdstnetworks_errs.WriteData(f, params)
	} else {
		WriteSpace(f, "packet-counts-and-bytes")
	}

	// Interfaces
	params["tab"] = "interfaces"
	params["valueformat"] = "number"

	params["title"] = "Incoming interface in original direction"
	if translate_interfaces {
		params["description"] = "Index of the interface that the packets are received from in the original session direction. Provided output of `diagnose netlink interface list` command is used to show interface's name."
		params["transform"] = transform_text
	} else {
		params["description"] = "Index of the interface that the packets are received from in the original session direction. To translate the index to name, output of `diagnose netlink interface list` command is needed."
		params["transform"] = nil
	}
	interfaces_org_in.WriteData(f, params)

	params["title"] = "Outgoing interface in original direction"
	if translate_interfaces {
		params["description"] = "Index of the interface that the packets are sent to in the original session direction. Provided output of `diagnose netlink interface list` command is used to show interface's name."
		params["transform"] = transform_text
	} else {
		params["description"] = "Index of the interface that the packets are sent to in the original session direction. To translate the index to name, output of `diagnose netlink interface list` command is needed."
		params["transform"] = nil
	}
	interfaces_org_out.WriteData(f, params)

	WriteSpace(f, "interfaces")

	params["title"] = "Incoming interface in reverse direction"
	if translate_interfaces {
		params["description"] = "Index of the interface that the packets are received from in the reverse session direction. To translate the index to name, output of `diagnose netlink interface list` command is needed."
		params["transform"] = transform_text
	} else {
		params["description"] = "Index of the interface that the packets are received from in the reverse session direction. Provided output of `diagnose netlink interface list` command is used to show interface's name."
		params["transform"] = nil
	}
	interfaces_rev_in.WriteData(f, params)

	params["title"] = "Outgoing interface in reverse direction"
	if translate_interfaces {
		params["description"] = "Index of the interface that the packets are sent to in the reverse session direction. To translate the index to name, output of `diagnose netlink interface list` command is needed."
		params["transform"] = transform_text
	} else {
		params["description"] = "Index of the interface that the packets are sent to in the reverse session direction. Provided output of `diagnose netlink interface list` command is used to show interface's name."
		params["transform"] = nil
	}
	interfaces_rev_out.WriteData(f, params)

	// Next hops 
	params["tab"] = "next-hops"
	params["transform"] = transform_ip
	params["valueformat"] = "number"
	params["title"] = "Next hop in original direction"
	params["description"] = "IP address of the next hop in the original session direction. This should match the routing table."
	nexthop_org.WriteData(f, params)
	params["title"] = "Next hop in reverse direction"
	params["description"] = "IP address of the next hop in the reverse session direction. This should reversely match the routing table."
	nexthop_rev.WriteData(f, params)

	// Durations & timeouts
	params["tab"] = "times"

	params["title"] = "Session time"
	params["description"] = "How long the session is already active."
	params["transform"] = transform_lifetime
	params["valueformat"] = "number"
	params["sortByKey"] = true
	durations.WriteData(f, params)

	params["title"] = "Session TTLs"
	params["description"] = "Initial TTLs configured for the sessions. Already closed sessions are of course not included."
	params["showOthers"] = true
	params["transform"] = nil
	params["valueformat"] = "number"
	params["sortByKey"] = false
	ttls.WriteData(f, params)

	// Offloading
	params["tab"] = "offload"

	params["title"] = "NPU offload"
	params["description"] = "Sessions offloaded to NPU hardware. Session can be fully offloaded in both directions or artially just in one direction (or not offloaded at all). Partial offload for UDP is usually caused by the traffic flowing only in a single direction."
	params["showOthers"] = false
	params["transform"] = transform_offload
	params["valueformat"] = "number"
	params["sortByKey"] = false
	offload_npu.WriteData(f, params)

	params["title"] = "nTurbo offload"
	params["description"] = "Sessions accelerated by nTurbo hardware. Eleigible sessions are those that would be offloaded if they didn't have UTM profiles and they have flow-based UTM profile configured."
	params["showOthers"] = false
	params["transform"] = transform_offload
	params["valueformat"] = "number"
	params["sortByKey"] = false
	offload_nturbo.WriteData(f, params)

	WriteSpace(f, "offload")

	params["title"] = "NPU offload fail generic"
	params["description"] = "The reason why the session could not be offloaded to NPU. Only sessions with non-empty field shown."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	offload_fail.WriteData(f, params)

	params["title"] = "NPU offload fail in forward direction"
	params["description"] = "The reason why the session could not be offloaded to NPU. Displayed as combined reason from kernel / driver."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	offload_fail_org.WriteData(f, params)

	params["title"] = "NPU offload fail in reverse direction"
	params["description"] = "The reason why the session could not be offloaded to NPU. Displayed as combined reason from kernel / driver."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	offload_fail_rev.WriteData(f, params)

	// Tunnels
	params["tab"] = "tunnels"

	params["title"] = "Incoming from tunnel"
	params["description"] = "Sessions traffic received inside a tunnel."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	tunnels_in.WriteData(f, params)

	params["title"] = "Outgoing to tunnel"
	params["description"] = "Sessions that send traffic through a tunnel."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	tunnels_out.WriteData(f, params)

	// Shapers
	params["tab"] = "shapers"

	params["title"] = "Shaper in original direction"
	params["description"] = "Number of session with traffic shaper applied on the original direction."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	shapers_org.WriteData(f, params)

	params["title"] = "Shaper in reverse direction"
	params["description"] = "Number of session with traffic shaper applied on the reverse direction."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	shapers_rev.WriteData(f, params)

	params["title"] = "Per-IP shaper"
	params["description"] = "Number of session with traffic shaper applied per IP."
	params["showOthers"] = true
	params["transform"] = transform_text
	params["valueformat"] = "number"
	params["sortByKey"] = false
	shapers_perip.WriteData(f, params)

	// Source NAT
	params["tab"] = "snat"

	params["title"] = "IP used as source NAT"
	params["description"] = "IP address that FortiGate selected as the source address when applying source NAT."
	params["showOthers"] = true
	params["transform"] = func(s interface{})(string) {
		if s.(uint32) == 0 { return "No source NAT"
		} else { return transform_ip(s) }
	}
	params["valueformat"] = "number"
	params["sortByKey"] = false
	snat_ip.WriteData(f, params)

	params["title"] = "Port used as source NAT"
	params["description"] = "Port number that FortiGate selected as the source address when applying source NAT."
	params["showOthers"] = true
	params["transform"] = func(s interface{})(string) {
		port := s.(uint16)
		if port == 0 { return "No source NAT"
		} else { return fmt.Sprintf("%d", port) }
	}
	params["valueformat"] = "number"
	params["sortByKey"] = false
	snat_port.WriteData(f, params)

	if use_complex_matching {
		params["title"] = "IP/port combination used as source NAT"
		params["description"] = "IP address and port number that FortiGate selected as the source address when applying source NAT."
		params["showOthers"] = true
		params["transform"] = func(s interface{})(string) {
			port := s.(uint64)
			if port == 0 { return "No source NAT"
			} else { return transform_ipport(s) }
		}
		params["valueformat"] = "number"
		params["sortByKey"] = false
		snat_ipport.WriteData(f, params)
	} else {
		WriteSpace(f, "snat")
	}

	// Session states
	params["tab"] = "session-states"

	params["title"] = "TCP client to FortiGate session state"
	params["description"] = "TCP session state for the 'left' session - the session between client and FortiGate."
	params["showOthers"] = true
	params["transform"] = transform_tcp_session_state
	params["valueformat"] = "number"
	params["sortByKey"] = false
	tcp_sstateL.WriteData(f, params)

	params["title"] = "TCP FortiGate to server session state"
	params["description"] = "TCP session state for the 'right' session - the session between FortiGate and server."
	params["showOthers"] = true
	params["transform"] = transform_tcp_session_state
	params["valueformat"] = "number"
	params["sortByKey"] = false
	tcp_sstateR.WriteData(f, params)

	params["title"] = "TCP combined session state"
	params["description"] = "TCP session state for both 'left' and 'right' sessions."
	params["showOthers"] = true
	params["transform"] = transform_tcp_combined_session_state
	params["valueformat"] = "number"
	params["sortByKey"] = false
	tcp_sstateLR.WriteData(f, params)

	params["title"] = "UDP client to FortiGate session state"
	params["description"] = "UDP session state for the 'left' session - the session between client and FortiGate."
	params["showOthers"] = true
	params["transform"] = transform_udp_session_state
	params["valueformat"] = "number"
	params["sortByKey"] = false
	udp_sstateL.WriteData(f, params)

	params["title"] = "UDP FortiGate to server session state"
	params["description"] = "UDP session state for the 'right' session - the session between FortiGate and server."
	params["showOthers"] = true
	params["transform"] = transform_udp_session_state
	params["valueformat"] = "number"
	params["sortByKey"] = false
	udp_sstateR.WriteData(f, params)

	params["title"] = "UDP combined session state"
	params["description"] = "UDP session state for both 'left' and 'right' sessions."
	params["showOthers"] = true
	params["transform"] = transform_udp_combined_session_state
	params["valueformat"] = "number"
	params["sortByKey"] = false
	udp_sstateLR.WriteData(f, params)

	// During this run we saved everything to `current` object, but without the following
	// code, during the next run this `current` object will be override (because data is
	// appended to the .js file. Therefore push the current `current` object to the 
	// `foset` array. With this, the `foset` array always contains all runs info from
	// the .js file.
	fmt.Fprintf(f, "if (typeof foset === 'undefined') { var foset = [] }\n")
	fmt.Fprintf(f, "foset.push(current)\n")

	// 
	f.Close()
}

// Create directory structure
func createDirectory(basedir string) {
	var err error

	err = os.MkdirAll(basedir, 0755)
	if err != nil {
		log.Criticalf("Unable to create base directory: %s", err)
		os.Exit(100)
	}

	for _, name := range AssetNames() {
		// just for sure split and join path using architecture speecific separator
		resource := path.Join(strings.Split(name, "/")...)

		// create sub-directory for each file
		err = os.MkdirAll(path.Join(basedir, path.Dir(resource)), 0755)
		if err != nil {
			log.Criticalf("Unable to create nested directory: %s", err)
			os.Exit(100)
		}

		// create file
		f, err := os.Create(path.Join(basedir, resource))
		if err != nil {
			log.Criticalf("Unable to create file: %s", err)
			os.Exit(100)
		}

		s, _ := Asset(name)
		_, err = f.Write(s)
		if err != nil {
			log.Criticalf("Unable to write file: %s", err)
			os.Exit(100)
		}

		f.Close()
	}
}

// Local functions
func getNetwork(ip net.IP, mask uint32) uint32 {
	ip  = ip.To4()
	num := binary.BigEndian.Uint32(ip)
	num &= mask
	return num
}

func initDictTCPState() {
	dict_tcp_session_state = make(map[uint8]string)
	dict_tcp_session_state[0] = "NONE"
	dict_tcp_session_state[1] = "ESTABLISHED"
	dict_tcp_session_state[2] = "SYN_SENT"
	dict_tcp_session_state[3] = "SYN_RECV"
	dict_tcp_session_state[4] = "FIN_WAIT1"
	dict_tcp_session_state[5] = "FIN_WAIT2"
	dict_tcp_session_state[6] = "TIME_WAIT"
	dict_tcp_session_state[7] = "CLOSE"
	dict_tcp_session_state[8] = "CLOSE_WAIT"
	dict_tcp_session_state[9] = "LAST_ACK"
	dict_tcp_session_state[10] = "LISTEN"
	dict_tcp_session_state[11] = "CLOSING"
}

func initDictUDPState() {
	dict_udp_session_state = make(map[uint8]string)
	dict_udp_session_state[0] = "UNSEEN"
	dict_udp_session_state[1] = "SEEN"
}
