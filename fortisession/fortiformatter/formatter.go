// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

// Print the session parameters using the user provided format string.
package fortiformatter

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"sort"
	"net"
	"strconv"
	"foset/fortisession"
	"github.com/juju/loggo"
	"os"
)

var log loggo.Logger

// InitLog initializes the logging system with the object passed from the calling program. 
// It is expected that this object is already initialized with loggo.GetLogger()
// function or something similar.
func InitLog(l loggo.Logger) { log = l }

type formatterParameter int
const (
	fp_serial    formatterParameter = iota
	fp_proto
	fp_src_ip
	fp_src_port
	fp_dst_ip
	fp_dst_port

	fp_state_l
	fp_state_r
	fp_duration
	fp_expire
	fp_timeout
	fp_det

	fp_sdap
	fp_sap
	fp_dap
	fp_nap
	fp_sa
	fp_sp
	fp_da
	fp_dp
	fp_na
	fp_np

	fp_rate_rx
	fp_rate_tx
	fp_rate_sum

	fp_npuflag_o
	fp_npuflag_r
	fp_offload_o
	fp_offload_r
	fp_nturbo_o
	fp_nturbo_r
	fp_innpu_o
	fp_innpu_f
	fp_outnpu_o
	fp_outnpu_f

	fp_offload_no
	fp_offload_fail_ko
	fp_offload_fail_kr
	fp_offload_fail_do
	fp_offload_fail_dr

	fp_count_op
	fp_count_ob
	fp_count_oe
	fp_count_rp
	fp_count_rb
	fp_count_re
	fp_count_o
	fp_count_r

	fp_policy
	fp_vdom

	fp_plain
	fp_newline

	fp_ha_id
	fp_helper
	fp_shaping_policy_id
	fp_tunnel_in
	fp_tunnel_out
	fp_tunnels

	fp_state

	fp_shaper_o
	fp_shaper_r
	fp_shaper_ip

	fp_mac_i
	fp_mac_o

	fp_iface_in_o
	fp_iface_out_o
	fp_iface_in_r
	fp_iface_out_r
	fp_next_hop_o
	fp_next_hop_r
	fp_path_o
	fp_path_r

	fp_auth_user
	fp_auth_server
	fp_auth_info

	fp_custom
)

type Formatter struct {
	str    string
	params []formatterParameter
	mods   []string
	form   []string
	empty  string                // replacement for empty strings
}

// Init initializes the Formatter with the format string given as parameter.
// The `request` parameter is a pointer to SessionDataRequest where the Init
// will set the fields it needs (based on the format string) to `true`.
// It will never set any field to `false`.
//
// Init returns the `Formatter` struct that is used to format any number of
// sessions.
//
// Format string description can be found in "output_format.md" file in this
// directory.
func Init(format string, request *fortisession.SessionDataRequest) (*Formatter, error) {
	f     := Formatter{empty: "-"}
	re    := regexp.MustCompile("\\${([^:]+?)(:.*?)?(\\|.*?)?}")
	format = f.demacro(format)

	var name, form, mod, before, after string
	after = format

	for {
		parts := re.FindStringSubmatchIndex(after)
		if len(parts) == 0 { break }

		name = after[parts[2]:parts[3]]
		if parts[4] == -1 && parts[5] == -1 {
			form = ""
		} else {
			form = after[parts[4]+1:parts[5]]
		}
		if parts[6] == -1 && parts[7] == -1 {
			mod = ""
		} else {
			mod = after[parts[6]+1:parts[7]]
		}
		before = after[:parts[0]]
		after  = after[parts[1]:]

		if form == "" {
			if name == "serial" { form = "x"
			} else if name == "sp" { form = "d"
			} else if name == "dp" { form = "d"
			} else if name == "np" { form = "d"
			} else if name == "policy" { form = "d"
			} else if name == "vdom" { form = "d"
			} else if name == "haid" { form = "d"
			} else if name == "duration" { form = "d"
			} else if name == "expire" { form = "d"
			} else if name == "timeout" { form = "d"
			} else if name == "npuflag[o]" { form = "#x"
			} else if name == "npuflag[r]" { form = "#x"
			} else if strings.HasPrefix(name, "iface[") { form = "d"
			} else if name == "authinfo" { form = "d"
			} else { form = "s" }
		}

		f.str += before + "%" + form

		if name == "serial" {
			f.params = append(f.params, fp_serial)
			request.Serial = true
		} else if name == "proto" || name == "protocol" {
			f.params = append(f.params, fp_proto)
			request.Basics = true
		} else if name == "state[l]" {
			f.params = append(f.params, fp_state_l)
			request.Basics = true
		} else if name == "state[r]" {
			f.params = append(f.params, fp_state_r)
			request.Basics = true
		} else if name == "duration" {
			f.params = append(f.params, fp_duration)
			request.Basics = true
		} else if name == "expire" {
			f.params = append(f.params, fp_expire)
			request.Basics = true
		} else if name == "timeout" {
			f.params = append(f.params, fp_timeout)
			request.Basics = true
		} else if name == "det" {
			f.params = append(f.params, fp_det)
			request.Basics = true
		} else if name == "sa" {
			f.params = append(f.params, fp_sa)
			request.Hooks = true
		} else if name == "da" {
			f.params = append(f.params, fp_da)
			request.Hooks = true
		} else if name == "na" {
			f.params = append(f.params, fp_na)
			request.Hooks = true
		} else if name == "sp" {
			f.params = append(f.params, fp_sp)
			request.Hooks = true
		} else if name == "dp" {
			f.params = append(f.params, fp_dp)
			request.Hooks = true
		} else if name == "np" {
			f.params = append(f.params, fp_np)
			request.Hooks = true
		} else if name == "sap" {
			f.params = append(f.params, fp_sap)
			request.Hooks = true
		} else if name == "dap" {
			f.params = append(f.params, fp_dap)
			request.Hooks = true
		} else if name == "nap" {
			f.params = append(f.params, fp_nap)
			request.Hooks = true
		} else if name == "sdap" {
			f.params = append(f.params, fp_sdap)
			request.Hooks = true
		} else if name == "rate[u]" {
			f.params = append(f.params, fp_rate_tx)
			request.Rate = true
		} else if name == "rate[d]" {
			f.params = append(f.params, fp_rate_rx)
			request.Rate = true
		} else if name == "rate[sum]" {
			f.params = append(f.params, fp_rate_sum)
			request.Rate = true
		} else if name == "npuflag[o]" {
			f.params = append(f.params, fp_npuflag_o)
			request.Npu = true
		} else if name == "npuflag[r]" {
			f.params = append(f.params, fp_npuflag_r)
			request.Npu = true
		} else if name == "offload[o]" {
			f.params = append(f.params, fp_offload_o)
			request.Npu = true
		} else if name == "offload[r]" {
			f.params = append(f.params, fp_offload_r)
			request.Npu = true
		} else if name == "nturbo[o]" {
			f.params = append(f.params, fp_nturbo_o)
			request.Npu = true
		} else if name == "nturbo[r]" {
			f.params = append(f.params, fp_nturbo_r)
			request.Npu = true
		} else if name == "innpu[o]" {
			f.params = append(f.params, fp_innpu_o)
			request.Npu = true
		} else if name == "innpu[f]" {
			f.params = append(f.params, fp_innpu_f)
			request.Npu = true
		} else if name == "outnpu[o]" {
			f.params = append(f.params, fp_outnpu_o)
			request.Npu = true
		} else if name == "outnpu[f]" {
			f.params = append(f.params, fp_outnpu_f)
			request.Npu = true
		} else if name == "nooff[no]" {
			f.params = append(f.params, fp_offload_no)
			request.NpuError = true
		} else if name == "nooff[ko]" {
			f.params = append(f.params, fp_offload_fail_ko)
			request.NpuError = true
		} else if name == "nooff[kr]" {
			f.params = append(f.params, fp_offload_fail_kr)
			request.NpuError = true
		} else if name == "nooff[do]" {
			f.params = append(f.params, fp_offload_fail_do)
			request.NpuError = true
		} else if name == "nooff[dr]" {
			f.params = append(f.params, fp_offload_fail_dr)
			request.NpuError = true
		} else if name == "count[op]" {
			f.params = append(f.params, fp_count_op)
			request.Stats = true
		} else if name == "count[ob]" {
			f.params = append(f.params, fp_count_ob)
			request.Stats = true
		} else if name == "count[oe]" {
			f.params = append(f.params, fp_count_oe)
			request.Stats = true
		} else if name == "count[rp]" {
			f.params = append(f.params, fp_count_rp)
			request.Stats = true
		} else if name == "count[rb]" {
			f.params = append(f.params, fp_count_rb)
			request.Stats = true
		} else if name == "count[re]" {
			f.params = append(f.params, fp_count_re)
			request.Stats = true
		} else if name == "count[o]" {
			f.params = append(f.params, fp_count_o)
			request.Stats = true
		} else if name == "count[r]" {
			f.params = append(f.params, fp_count_r)
			request.Stats = true
		} else if name == "vdom" {
			f.params = append(f.params, fp_vdom)
			request.Policy = true
		} else if name == "policy" {
			f.params = append(f.params, fp_policy)
			request.Policy = true
		} else if name == "helper" {
			f.params = append(f.params, fp_helper)
			request.Other = true
		} else if name == "state" {
			f.params = append(f.params, fp_state)
			request.States = true
		} else if name == "haid" {
			f.params = append(f.params, fp_ha_id)
			request.Other = true
		} else if name == "shapingpolicy" {
			f.params = append(f.params, fp_shaping_policy_id)
			request.Other = true
		} else if name == "tunnel[i]" {
			f.params = append(f.params, fp_tunnel_in)
			request.Other = true
		} else if name == "tunnel[o]" {
			f.params = append(f.params, fp_tunnel_out)
			request.Other = true
		} else if name == "tunnels" {
			f.params = append(f.params, fp_tunnels)
			request.Other = true
		} else if name == "shaper[o]" {
			f.params = append(f.params, fp_shaper_o)
			request.Shaping = true
		} else if name == "shaper[r]" {
			f.params = append(f.params, fp_shaper_o)
			request.Shaping= true
		} else if name == "shaper[ip]" {
			f.params = append(f.params, fp_shaper_ip)
			request.Shaping = true
		} else if name == "mac[i]" || name == "mac[src]" {
			f.params = append(f.params, fp_mac_i)
			request.Macs = true
		} else if name == "mac[o]" || name == "mac[dst]" {
			f.params = append(f.params, fp_mac_o)
			request.Macs = true
		} else if name == "iface[oi]" || name == "iface[io]" {
			f.params = append(f.params, fp_iface_in_o)
			request.Interfaces = true
		} else if name == "iface[oo]" {
			f.params = append(f.params, fp_iface_out_o)
			request.Interfaces = true
		} else if name == "iface[ri]" || name == "iface[ir]" {
			f.params = append(f.params, fp_iface_in_r)
			request.Interfaces = true
		} else if name == "iface[ro]" || name == "iface[or]" {
			f.params = append(f.params, fp_iface_out_r)
			request.Interfaces = true
		} else if name == "nexthop[o]" || name == "nh[o]" {
			f.params = append(f.params, fp_next_hop_o)
			request.Interfaces = true
		} else if name == "nexthop[r]" || name == "nh[r]" {
			f.params = append(f.params, fp_next_hop_r)
			request.Interfaces = true
		} else if name == "patho" {
			f.params = append(f.params, fp_path_o)
			request.Interfaces = true
		} else if name == "pathr" {
			f.params = append(f.params, fp_path_r)
			request.Interfaces = true
		} else if name == "user" {
			f.params = append(f.params, fp_auth_user)
			request.Auth = true
		} else if name == "authserver" {
			f.params = append(f.params, fp_auth_server)
			request.Auth = true
		} else if name == "authinfo" {
			f.params = append(f.params, fp_auth_info)
			request.Auth = true
		} else if name == "custom" {
			f.params = append(f.params, fp_custom)
			request.Custom = true
		} else if name == "plain" {
			f.params = append(f.params, fp_plain)
			request.Plain = true
		} else if name == "newline" {
			f.params = append(f.params, fp_newline)
		} else { return nil, fmt.Errorf("Unknown format variable \"%s\"", name) }

		f.mods = append(f.mods, mod)
		f.form = append(f.form, form)
	}
	f.str += after

	log.Tracef("Formatter string: \"%s\"", f.str[:len(f.str)])
	return &f, nil
}

// Format returns string based on the format passed to Init function and 
// the session data passed to Format function.
func (f *Formatter) Format(session *fortisession.Session) string {
	var params []interface{}

	for index, p := range f.params {
		if p == fp_serial                   { params = append(params, session.Serial)
		} else if p == fp_plain             { params = append(params, f.format_plain(session.Plain))
		} else if p == fp_newline           { params = append(params, "\n")
		} else if p == fp_ha_id             { params = append(params, session.Other.HAid)
		} else if p == fp_helper            { params = append(params, f.stringOrDash(session.Other.Helper))
		} else if p == fp_tunnel_in         { params = append(params, f.stringOrDash(session.Other.Tunnel_in))
		} else if p == fp_tunnel_out        { params = append(params, f.stringOrDash(session.Other.Tunnel_out))
		} else if p == fp_tunnels           {
			params = append(params, fmt.Sprintf("%s->%s", f.stringOrDash(session.Other.Tunnel_in), f.stringOrDash(session.Other.Tunnel_out)))
		} else if p == fp_state     { params = append(params, f.format_state(session.States, f.mods[index]))
		} else if p == fp_duration  { params = append(params, session.Basics.Duration)
		} else if p == fp_expire    { params = append(params, session.Basics.Expire)
		} else if p == fp_timeout   { params = append(params, session.Basics.Timeout)
		} else if p == fp_det       {
			params = append(params, fmt.Sprintf("%6d/%-5d(%d)", session.Basics.Duration, session.Basics.Expire, session.Basics.Timeout))
		} else if p == fp_count_op  { params = append(params, f.format_stats(session.Stats.Packets_org, session.Stats.Valid_org))
		} else if p == fp_count_ob  { params = append(params, f.format_stats(session.Stats.Bytes_org, session.Stats.Valid_org))
		} else if p == fp_count_oe  { params = append(params, f.format_stats(session.Stats.Errors_org, session.Stats.Valid_org))
		} else if p == fp_count_o   {
			op := f.format_stats(session.Stats.Packets_org, session.Stats.Valid_org)
			ob := f.format_stats(session.Stats.Bytes_org, session.Stats.Valid_org)
			oe := f.format_stats(session.Stats.Errors_org, session.Stats.Valid_org)
			params = append(params, fmt.Sprintf("%s/%s/%s", ob, op, oe))
		} else if p == fp_count_r   {
			rp := f.format_stats(session.Stats.Packets_rev, session.Stats.Valid_rev)
			rb := f.format_stats(session.Stats.Bytes_rev, session.Stats.Valid_rev)
			re := f.format_stats(session.Stats.Errors_rev, session.Stats.Valid_rev)
			params = append(params, fmt.Sprintf("%s/%s/%s", rb, rp, re))
		} else if p == fp_count_rp  { params = append(params, f.format_stats(session.Stats.Packets_rev, session.Stats.Valid_rev))
		} else if p == fp_count_rb  { params = append(params, f.format_stats(session.Stats.Bytes_rev, session.Stats.Valid_rev))
		} else if p == fp_count_re  { params = append(params, f.format_stats(session.Stats.Errors_rev, session.Stats.Valid_rev))
		} else if p == fp_shaping_policy_id {
			if strings.Contains(f.form[index], "s") {
				params = append(params, f.format_shaping_policy(session.Other.ShapingPolicyId))
			} else {
				params = append(params, session.Other.ShapingPolicyId)
			}
		} else if p == fp_policy    {
			if strings.Contains(f.form[index], "s") {
				params = append(params, f.format_policy(session.Policy.Id))
			} else {
				params = append(params, session.Policy.Id)
			}
		} else if p == fp_vdom      {
			if strings.Contains(f.form[index], "s") {
				params = append(params, f.format_vdom(session.Policy.Vdom, f.mods[index]))
			} else {
				params = append(params, session.Policy.Vdom)
			}
		} else if p == fp_proto {
			if strings.Contains(f.form[index], "s") {
				params = append(params, f.format_proto(session.Basics.Protocol))
			} else {
				params = append(params, session.Basics.Protocol)
			}
		} else if p == fp_state_l {
			if strings.Contains(f.form[index], "s") {
				params = append(params, f.format_proto_state(session.Basics.Protocol, session.Basics.StateL))
			} else {
				params = append(params, session.Basics.StateL)
			}
		} else if p == fp_state_r {
			if strings.Contains(f.form[index], "s") {
				params = append(params, f.format_proto_state(session.Basics.Protocol, session.Basics.StateR))
			} else {
				params = append(params, session.Basics.StateR)
			}
		} else if p == fp_npuflag_o { params = append(params, session.Npu.Flag_org)
		} else if p == fp_npuflag_r { params = append(params, session.Npu.Flag_rev)
		} else if p == fp_offload_o {
			if strings.Contains(f.form[index], "s") {
				if session.Npu.Offload_org == 0 { params = append(params, "N")
				} else { params = append(params, "Y") }
			} else {
				params = append(params, session.Npu.Offload_org)
			}
		} else if p == fp_offload_r {
			if strings.Contains(f.form[index], "s") {
				if session.Npu.Offload_rev == 0 { params = append(params, "N")
				} else { params = append(params, "Y") }
			} else {
				params = append(params, session.Npu.Offload_rev)
			}
		} else if p == fp_nturbo_o {
			if strings.Contains(f.form[index], "s") {
				if session.Npu.Nturbo_org == 0 { params = append(params, "N")
				} else { params = append(params, "Y") }
			} else {
				params = append(params, session.Npu.Nturbo_org)
			}
		} else if p == fp_nturbo_r {
			if strings.Contains(f.form[index], "s") {
				if session.Npu.Nturbo_rev == 0 { params = append(params, "N")
				} else { params = append(params, "Y") }
			} else {
				params = append(params, session.Npu.Nturbo_rev)
			}
		} else if p == fp_innpu_o { params = append(params, f.format_npu(session.Npu.InNpu_org, session.Npu.InNpu_org_valid))
		} else if p == fp_innpu_f { params = append(params, f.format_npu(session.Npu.InNpu_fwd, session.Npu.InNpu_fwd_valid))
		} else if p == fp_outnpu_o { params = append(params, f.format_npu(session.Npu.OutNpu_org, session.Npu.OutNpu_org_valid))
		} else if p == fp_outnpu_f { params = append(params, f.format_npu(session.Npu.OutNpu_fwd, session.Npu.OutNpu_fwd_valid))
		} else if p == fp_sdap {
			src_ip, src_port, dst_ip, dst_port, _, _, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s:%d->%s:%d", src_ip.String(), src_port, dst_ip.String(), dst_port))
		} else if p == fp_sap {
			src_ip, src_port, _, _, _, _, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s:%d", src_ip.String(), src_port))
		} else if p == fp_dap {
			_, _, dst_ip, dst_port, _, _, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s:%d", dst_ip.String(), dst_port))
		} else if p == fp_nap {
			_, _, _, _, nat_ip, nat_port, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s:%d", nat_ip.String(), nat_port))
		} else if p == fp_sa {
			src_ip, _, _, _, _, _, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s", f.format_address(src_ip, f.mods[index])))
		} else if p == fp_da {
			_, _, dst_ip, _, _, _, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s", f.format_address(dst_ip, f.mods[index])))
		} else if p == fp_na {
			_, _, _, _, nat_ip, _, _ := session.GetPeers()
			params = append(params, fmt.Sprintf("%s", f.format_address(nat_ip, f.mods[index])))
		} else if p == fp_sp {
			_, src_port, _, _, _, _, _ := session.GetPeers()
			params = append(params, src_port)
		} else if p == fp_dp {
			_, _, _, dst_port, _, _, _ := session.GetPeers()
			params = append(params, dst_port)
		} else if p == fp_np {
			_, _, _, _, _, nat_port, _ := session.GetPeers()
			params = append(params, nat_port)
		} else if p == fp_rate_rx {
			rate_int, rate_str, rate_float := f.format_rate(session.Rate.Rx_Bps, f.mods[index])
			if strings.Contains(f.form[index], "s") {
				params = append(params, rate_str)
			} else if strings.Contains(f.form[index], "f") {
				params = append(params, rate_float)
			} else {
				params = append(params, rate_int)
			}
		} else if p == fp_rate_tx {
			rate_int, rate_str, rate_float := f.format_rate(session.Rate.Tx_Bps, f.mods[index])
			if strings.Contains(f.form[index], "s") {
				params = append(params, rate_str)
			} else if strings.Contains(f.form[index], "f") {
				params = append(params, rate_float)
			} else {
				params = append(params, rate_int)
			}
		} else if p == fp_rate_sum {
			rate_int, rate_str, rate_float := f.format_rate(session.Rate.Rx_Bps+session.Rate.Tx_Bps, f.mods[index])
			if strings.Contains(f.form[index], "s") {
				params = append(params, rate_str)
			} else if strings.Contains(f.form[index], "f") {
				params = append(params, rate_float)
			} else {
				params = append(params, rate_int)
			}
		} else if p == fp_offload_no      { params = append(params, f.stringOrDash(session.NpuError.NoOffloadReason))
		} else if p == fp_offload_fail_ko { params = append(params, f.stringOrDash(session.NpuError.Kernel_org))
		} else if p == fp_offload_fail_kr { params = append(params, f.stringOrDash(session.NpuError.Kernel_rev))
		} else if p == fp_offload_fail_do { params = append(params, f.stringOrDash(session.NpuError.Driver_org))
		} else if p == fp_offload_fail_dr { params = append(params, f.stringOrDash(session.NpuError.Driver_rev))
		} else if p == fp_shaper_o        { params = append(params, f.stringOrDash(session.Shaping.Shaper_org))
		} else if p == fp_shaper_r        { params = append(params, f.stringOrDash(session.Shaping.Shaper_rev))
		} else if p == fp_shaper_ip       { params = append(params, f.stringOrDash(session.Shaping.Shaper_ip))
		} else if p == fp_mac_i           { params = append(params, f.stringOrDash(session.Macs.Src))
		} else if p == fp_mac_o           { params = append(params, f.stringOrDash(session.Macs.Dst))
		} else if p == fp_iface_in_o      { params = append(params, session.Interfaces.In_org)
		} else if p == fp_iface_out_o     { params = append(params, session.Interfaces.Out_org)
		} else if p == fp_iface_in_r      { params = append(params, session.Interfaces.In_rev)
		} else if p == fp_iface_out_r     { params = append(params, session.Interfaces.Out_rev)
		} else if p == fp_next_hop_o      { params = append(params, session.Interfaces.NextHop_org.String())
		} else if p == fp_next_hop_r      { params = append(params, session.Interfaces.NextHop_rev.String())
		} else if p == fp_path_o          {
			params = append(params, fmt.Sprintf("%3d->%-3d %15s", session.Interfaces.In_org, session.Interfaces.Out_org, session.Interfaces.NextHop_org.String()))
		} else if p == fp_path_r          {
			params = append(params, fmt.Sprintf("%3d->%-3d %15s", session.Interfaces.In_rev, session.Interfaces.Out_rev, session.Interfaces.NextHop_rev.String()))
		} else if p == fp_auth_user       { params = append(params, f.stringOrDash(session.Auth.User))
		} else if p == fp_auth_server     { params = append(params, f.stringOrDash(session.Auth.Profile))
		} else if p == fp_auth_info       { params = append(params, session.Auth.AuthInfo)
		} else if p == fp_custom          {
			value, exists := session.Custom[f.mods[index]]
			if strings.Contains(f.form[index], "s") {
				if !exists                  { params = append(params, f.stringOrDash(""))
				} else                      { params = append(params, f.stringOrDash(value.AsString())) }
			} else if strings.Contains(f.form[index], "f") {
				if !exists                  { params = append(params, float64(0))
				} else                      { params = append(params, value.AsFloat64()) }
			} else if strings.Contains(f.form[index], "x") {
				if !exists                  { params = append(params, uint64(0))
				} else                      { params = append(params, value.AsUint64()) }
			} else {
				if !exists                  { params = append(params, uint64(0))
				} else                      { params = append(params, value.AsUint64()) }
			}
		}
	}

	log.Tracef("Formatter params: %#f", params)
	return fmt.Sprintf(f.str, params...)
}

func (f *Formatter) demacro(format string) string {
	macros := make(map[string]string)
	macros["${default_basics}"]  = "${serial:08x}: ${vdom:3d}/${policy:-5s} ${proto:-4s} ${state[l]:11s}/${state[r]:-11s} ${sap:-21s} -> ${dap:-21s}"
	macros["${default_basic}"]  = macros["${default_basics}"]
	macros["${default_hw}"]      = "OFF(${offload[o]}/${offload[r]}), NTB(${nturbo[o]}/${nturbo[r]}) FLG(${npuflag[o]:#02x}/${npuflag[r]:#02x})"
	macros["${default_rate}"]    = "RATE(up:${rate[u]:15s}, down:${rate[d]:15s})"
	macros["${default_counts}"]  = "COUNTS(org:${count[o]:15s}, rev:${count[r]:15s})"
	macros["${default_count}"]   = macros["${default_count}"]
	macros["${default_time}"]    = "TIME(${det:-20s})"
	macros["${default_states}"]  = "${state:-60s|, }"
	macros["${default_nooff}"]   = "${nooff[ko]}/${nooff[kr]},${nooff[do]}/${nooff[dr]}"
	macros["${default_path}"]    = "INF(org: ${patho}, rev: ${pathr})"
	macros["${default_macs}"]    = "${mac[i]:17s} -> ${mac[o]:17s}"

	var ret string = format
	for key, value := range macros {
		ret = strings.Replace(ret, key, value, -1)
	}

	return ret
}

func (f *Formatter) format_address(addr net.IP, mod string) string {
	if len(mod) == 0 {
		return addr.String()
	}

	for _, m := range strings.Split(mod, ",") {
		if strings.HasPrefix(m, "mask:") {
			mask_len, err := strconv.ParseUint(m[5:], 10, 8)
			if err != nil {
				log.Criticalf("Cannot convert IP mask \"%s\"", m)
				os.Exit(100)
			}
			mask := net.CIDRMask(int(mask_len), 32)
			addr = addr.Mask(mask)
		}
	}

	return addr.String()
}

func (f *Formatter) format_rate(rate_Bps uint64, mod string) (uint64, string, float64) {
	var divide float64 = 1
	var text string

	if len(mod) == 0 { // auto
		if (rate_Bps*8) > 1000*1000*1000*1000 {
			mod = "tbps"
		} else if (rate_Bps*8) > 1000*1000*1000 {
			mod = "gbps"
		} else if (rate_Bps*8) > 1000*1000 {
			mod = "mbps"
		} else if (rate_Bps*8) > 1000 {
			mod = "kbps"
		} else {
			mod = "bps"
		}
	}

	if strings.HasPrefix(strings.ToLower(mod), "ki") {
		divide = 1024
		text   = "Ki"
		mod = mod[2:]
	} else if strings.HasPrefix(strings.ToLower(mod), "k") {
		divide = 1000
		text   = "K"
		mod = mod[1:]
	} else if strings.HasPrefix(strings.ToLower(mod), "mi") {
		divide = 1024*1024
		text   = "Mi"
		mod = mod[2:]
	} else if strings.HasPrefix(strings.ToLower(mod), "m") {
		divide = 1000*1000
		text   = "M"
		mod = mod[1:]
	} else if strings.HasPrefix(strings.ToLower(mod), "gi") {
		divide = 1024*1024*1024
		text   = "Gi"
		mod = mod[2:]
	} else if strings.HasPrefix(strings.ToLower(mod), "g") {
		divide = 1000*1000*1000
		text   = "G"
		mod = mod[1:]
	} else if strings.HasPrefix(strings.ToLower(mod), "ti") {
		divide = 1024*1024*1024*1024
		text   = "Ti"
		mod = mod[2:]
	} else if strings.HasPrefix(strings.ToLower(mod), "t") {
		divide = 1000*1000*1000*1000
		text   = "T"
		mod = mod[1:]
	}


	if strings.ToLower(mod) != "bps" {
		log.Criticalf("Unknown rate format")
		os.Exit(100)
	}

	if mod[0] == 'b' {
		divide /= 8
		text += "bps"
	} else if mod[0] == 'B' {
		text += "Bps"
	}

	var value float64 = float64(rate_Bps) / divide
	return uint64(math.Round(value)),
		fmt.Sprintf("%.3f %s", value, text),
		value
}

func (f *Formatter) format_shaping_policy(shaping_policy uint32) string {
	if shaping_policy == 0 {
		return f.stringOrDash("")
	} else {
		return fmt.Sprintf("%d", shaping_policy)
	}
}

func (f *Formatter) format_policy(policy uint32) string {
	if policy == 4294967295 {
		return "i"
	} else {
		return fmt.Sprintf("%d", policy)
	}
}

func (f *Formatter) format_vdom(vdom uint32, mod string) string {
	names := make(map[uint32]string)
	for _, part := range strings.Split(mod, ",") {
		if len(part) == 0 { continue }

		eq := strings.Index(part, "=")
		if eq == -1 {
			log.Criticalf("Invalid vdom name mapping format: %s", part)
			os.Exit(100)
		}

		num, err := strconv.ParseUint(part[:eq], 10, 32)
		if err != nil {
			log.Criticalf("Invalid vdom name index \"%s\": %s", part, err)
			os.Exit(100)
		}

		names[uint32(num)] = part[eq+1:]
	}

	//
	name, ok := names[vdom]
	if !ok {
		return fmt.Sprintf("%d", vdom)
	} else {
		return name
	}
}

func (f *Formatter) format_proto(proto uint16) string {
	if proto ==  1 { return "ICMP" }
	if proto ==  6 { return "TCP"  }
	if proto == 17 { return "UDP"  }
	if proto == 41 { return "IPv6" }
	if proto == 47 { return "GRE"  }
	if proto == 50 { return "ESP"  }
	return fmt.Sprintf("%d", proto)
}

func (f *Formatter) format_plain(plain string) string {
	var start int = 0
	var end   int = len(plain)-1

	for plain[start] == '\n' { start += 1 }
	for plain[end]   == '\n' { end   -= 1 }

	return plain[start:end+1]
}

func (f *Formatter) format_npu(npu uint8, valid bool) string {
	if !valid {
		return f.stringOrDash("")
	} else {
		return fmt.Sprintf("%d", npu)
	}
}

func (f *Formatter) format_stats(stat uint64, valid bool) string {
	if !valid {
		return "?"
	} else {
		return fmt.Sprintf("%d", stat)
	}
}

func (f *Formatter) format_state(states []fortisession.State, mod string) string {
	var delimiter string
	conv      := make([]string, len(states))

	d := strings.Index(mod, "|")
	if d != -1 {
		delimiter = mod[:d]
		mod       = mod[d+1:]
	} else {
		delimiter = mod
		mod       = ""
	}

	if len(delimiter) == 0 { delimiter = "," }

	for i, s := range states {
		conv[i] = string(s)
	}

	sort_it   := false
	filter_it := make([]string, 0)

	for _, xm := range strings.Split(mod, ";") {
		if xm == "sort" {
			sort_it = true

		} else if strings.HasPrefix(xm, "filter:") {
			filter_it = append(filter_it, strings.Split(xm[7:], ",")...)

		} else if len(xm) == 0 {
			continue

		} else {
			log.Criticalf("Unknown state modifier \"%s\"", xm)
			os.Exit(100)
		}
	}

	if sort_it {
		sort.Strings(conv)
	}

	newconv := make([]string, 0)
	for _, ov := range conv {
		found := false

		for _, fv := range filter_it {
			if ov == fv {
				found = true
				break
			}
		}

		if found || len(filter_it) == 0 {
			newconv = append(newconv, ov)
		}
	}

	return strings.Join(newconv, delimiter)
}

func (f *Formatter) format_proto_state(proto uint16, state uint8) string {
	if proto == 6 {
		// this is from Linux kernel but fortios seems to use different numbers
		/*
		if state ==  0 { return "NONE" }
		if state ==  1 { return "ESTABLISHED" }
		if state ==  2 { return "SYN_SENT" }
		if state ==  3 { return "SYN_RECV" }
		if state ==  4 { return "FIN_WAIT1" }
		if state ==  5 { return "FIN_WAIT2" }
		if state ==  6 { return "TIME_WAIT" }
		if state ==  7 { return "CLOSE" }
		if state ==  8 { return "CLOSE_WAIT" }
		if state ==  9 { return "LAST_ACK" }
		if state == 10 { return "LISTEN" }
		if state == 11 { return "CLOSING" }
		*/
		// from https://kb.fortinet.com/kb/viewContent.do?externalId=FD30042
		// partially validated (at least matches better than above)
		if state ==  0 { return "NONE" }
		if state ==  1 { return "ESTABLISHED" }
		if state ==  2 { return "SYN_SENT" }
		if state ==  3 { return "SYN_RECV" }
		if state ==  4 { return "FIN_WAIT" }
		if state ==  5 { return "TIME_WAIT" }
		if state ==  6 { return "CLOSE" }
		if state ==  7 { return "CLOSE_WAIT" }
		if state ==  8 { return "LAST_ACK" }
		if state ==  9 { return "LISTEN" }
		return "UNKNOWN"
	} else if proto == 17 {
		if state == 0 { return "SEEN" }
		if state == 1 { return "UNSEEN" }
		return "UNKNOWN"
	} else {
		return fmt.Sprintf("%d", state)
	}
}

func (f *Formatter) format_custom(custom map[string]string, mods string) (string, float64, int64) {
	if custom == nil { return f.stringOrDash(""), 0, 0 }

	v_string, exists := custom[mods]
	if !exists { return f.stringOrDash(""), 0, 0 }

	v_int, _   := strconv.ParseInt(v_string, 10, 64)
	v_float, _ := strconv.ParseFloat(v_string, 64)
	return v_string, v_float, v_int
}

func (f *Formatter) stringOrDash(i string) string {
	if len(i) == 0 { return f.empty } else { return i }
}

