// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

// Test whether the session parameters match the user provided filter (conditions).
package forticonditioner

import (
	"strings"
	"regexp"
	"fmt"
	"net"
	"strconv"
	"foset/fortisession"
	"foset/fortisession/multivalue"
	"foset/fortisession/fortiformatter"
	"github.com/juju/loggo"
	"os"
)

var log loggo.Logger

// InitLog initializes the logging system with the object passed from the calling program. 
// It is expected that this object is already initialized with loggo.GetLogger()
// function or something similar.
func InitLog(l loggo.Logger) {
	log = l
	fortiformatter.InitLog(log.Child("nested-formatter"))
}

type conditionerParameter int
const (
	cp_host            conditionerParameter = iota
	cp_shost
	cp_dhost
	cp_nhost
	cp_port
	cp_sport
	cp_dport
	cp_nport
	cp_policy
	cp_vdom
	cp_helper
	cp_state
	cp_status_l
	cp_status_r
	cp_status
	cp_proto
	cp_serial
	cp_npuflag_o
	cp_npuflag_r
	cp_npuflag
	cp_offload
	cp_offload_o
	cp_offload_r
	cp_nooff_no
	cp_nooff_ko
	cp_nooff_kr
	cp_nooff_do
	cp_nooff_dr
	cp_nturbo
	cp_nturbo_o
	cp_nturbo_r
	cp_innpu_o
	cp_innpu_f
	cp_outnpu_o
	cp_outnpu_f
	cp_rate_u
	cp_rate_d
	cp_rate
	cp_rate_sum
	cp_count_ob
	cp_count_op
	cp_count_oe
	cp_count_rb
	cp_count_rp
	cp_count_re
	cp_shapingpolicy
	cp_tunnel_in
	cp_tunnel_out
	cp_tunnel
	cp_shaper_o
	cp_shaper_r
	cp_shaper_ip
	cp_shaper
	cp_mac_i
	cp_mac_o
	cp_mac
	cp_iface_in_o
	cp_iface_out_o
	cp_iface_in_r
	cp_iface_out_r
	cp_iface
	cp_nexthop_o
	cp_nexthop_r
	cp_nexthop
	cp_user
	cp_auth_server
	cp_auth_info
	cp_custom
)

type Condition struct {
	sub     conditionOrExpression
}

type conditionOrExpression interface {
	isAnd()         bool
	isOr()          bool
	isExpression()  bool
}

type expression struct {
	lside      conditionerParameter
	operator   string
	rside      string
	negative   bool
	extra      string
	formatter  *fortiformatter.Formatter
}

func (expression) isAnd()         bool { return false }
func (expression) isOr()          bool { return false }
func (expression) isExpression()  bool { return true  }

type and struct {
	sub  []conditionOrExpression
}

func (and) isAnd()         bool { return true  }
func (and) isOr()          bool { return false }
func (and) isExpression()  bool { return false }

type or struct {
	sub  []conditionOrExpression
}

func (or) isAnd()         bool { return false }
func (or) isOr()          bool { return true  }
func (or) isExpression()  bool { return false }

// Init initializes the Condition with the filter string given as parameter.
// The `request` parameter is a pointer to SessionDataRequest where the Init
// will set the fields it needs (based on the filter string) to `true`.
// It will never set any field to `false`.
//
// Init returns the `Condition` struct that is used to check any number of
// sessions.
//
// For the format of `filter` string refer to "condition_format.md" file in this directory.
//
func Init(filter string, request *fortisession.SessionDataRequest) (*Condition) {
	var cond Condition
	c := simplify(parse(filter, request))
	cond.sub     = c
	return &cond
}

// Matches returns true if the given session matches the filter
// specified in the Init function. Otherwise it returns false.
func (cond *Condition) Matches(session *fortisession.Session) bool {
	return cond.match(cond.sub, session)
}

// DumpPretty is used for debugging. It returns string that describes
// the "decision tree" together with parsed parameters.
func (cond *Condition) DumpPretty() string {
	return dumpPretty(cond.sub, 0)
}

/*
 * Parsing condition strings
 */

func firstLevelNestingIndexes(cstr string) ([][]int) {
	var cnt, pos, start, open int

	ret := make([][]int, 0)

	for {
		pos = strings.IndexAny(cstr[start:], "()")
		if pos == -1 {
			break
		}

		if cstr[start+pos] == '(' {
			cnt += 1
			if cnt == 1 {
				open = start+pos
			}

		} else if cstr[start+pos] == ')' {
			cnt -= 1
			if cnt == 0 {
//				fmt.Printf("Excluding: \"%s\"\n", cstr[open:start+pos+1])
				ret = append(ret, []int{open, start+pos+1})
				open = -1
			}
		}

		start = start+pos+1
	}

	return ret
}

func isInRange(search []int, rng [][]int) (bool) {
	for _, r := range rng {
		if search[0] >= r[0] && search[1] < r[1] {
			return true
		}
	}

	return false
}

func splitBy(cstr string, delimiter string) ([]string) {
	excluded := firstLevelNestingIndexes(cstr)
//	fmt.Printf("Excluded indexes: %#v\n", excluded)

	var start int
	keep := make([][]int, 0)

	g := regexp.MustCompile("(?i)\\s" + delimiter + "\\s").FindAllStringIndex(cstr, -1)
	for _, e := range g {
		if isInRange(e, excluded) { continue }
		keep = append(keep, []int{start, e[0]})
		start = e[1]
	}
	keep = append(keep, []int{start, len(cstr)})

	var parts []string
	for _, e := range keep {
		part := strings.TrimSpace(cstr[e[0]:e[1]])
		if len(part) == 0 { continue }
		parts = append(parts, part)
	}

	return parts
}

func simplify(cond conditionOrExpression) (conditionOrExpression) {
	if cond.isExpression() {
		return cond

	} else if cond.isAnd() {
		s := cond.(and).sub
		if len(s) == 1 {
			return simplify(s[0])
		} else {
			var and and
			for _, ss := range s {
				and.sub = append(and.sub, simplify(ss))
			}
			return and
		}

	} else if cond.isOr() {
		s := cond.(or).sub
		if len(s) == 1 {
			return simplify(s[0])
		} else {
			var or or
			for _, ss := range s {
				or.sub = append(or.sub, simplify(ss))
			}
			return or
		}
	}

	// never get here
	return nil
}

func parse(cstr string, request *fortisession.SessionDataRequest) (conditionOrExpression) {
	var and and
	for _, and_str := range splitBy(cstr, "and") {
		var or or
		for _, or_str := range splitBy(and_str, "or") {
			or.sub = append(or.sub, makeExpression(or_str, request))
		}

		and.sub = append(and.sub, or)
	}
	return and
}

func makeExpression(expr string, request *fortisession.SessionDataRequest) (conditionOrExpression) {
	expr = strings.TrimSpace(expr)

	if expr[0] == '(' && expr[len(expr)-1] == ')' {
		return parse(expr[1:len(expr)-1], request)
	}

	var ret  expression

	// negative (or possitive - just for better looking conditions)
	if len(expr) >= 4 && strings.ToLower(expr)[:4] == "has " {
		expr = expr[4:]
	} else if len(expr) >= 3 && strings.ToLower(expr)[:3] == "is " {
		expr = expr[3:]
	}
	if len(expr) >= 4 && strings.ToLower(expr)[:4] == "not " {
		ret.negative = true
		expr = expr[4:]
	} else if len(expr) >= 3 && strings.ToLower(expr)[:3] == "no " {
		ret.negative = true
		expr = expr[3:]
	}

	// custom variable access
	if strings.HasPrefix(expr, "custom ") {
		s := strings.Index(expr[7:], " ")
		if s == -1 {
			ret.extra = expr[7:]
			expr      = "custom"
		} else {
			ret.extra = expr[7:7+s]
			expr      = "custom" + expr[7+s:]
		}
	}

	//
	test := regexp.MustCompile("[^ ]+")
	g := test.FindAllStringIndex(expr, 3)

	if len(g) == 0 {
		log.Criticalf("Unable to parse condition expression \"%s\"", expr)
		os.Exit(100)
	}

	if len(g) == 1 {
		ret.lside = convertLside(expr[g[0][0]:g[0][1]], request)

	} else if len(g) == 2 {
		ret.lside = convertLside(expr[g[0][0]:g[0][1]], request)
		ret.rside = expr[g[1][0]:g[1][1]]
	} else { // >= 3
		ret.lside    = convertLside(expr[g[0][0]:g[0][1]], request)
		ret.operator = expr[g[1][0]:g[1][1]]
		ret.rside    = expr[g[2][0]:]
	}

	// right side can also be formatter expression
	if len(ret.rside) > 2 && ret.rside[0] == '|' && ret.rside[len(ret.rside)-1] == '|' {
		var err error
		ret.formatter, err = fortiformatter.Init(ret.rside[1:len(ret.rside)-1], request)
		if err != nil {
			log.Criticalf("Cannot initialize nested formatter: %s", err)
			os.Exit(100)
		}
	}

	return ret
}

func convertLside(lside string, request *fortisession.SessionDataRequest) conditionerParameter {
	if lside == "host" {
		request.Hooks = true
		return cp_host
	} else if lside == "shost" {
		request.Hooks = true
		return cp_shost
	} else if lside == "dhost" {
		request.Hooks = true
		return cp_dhost
	} else if lside == "nhost" {
		request.Hooks = true
		return cp_nhost
	} else if lside == "port" {
		request.Hooks = true
		return cp_port
	} else if lside == "sport" {
		request.Hooks = true
		return cp_sport
	} else if lside == "dport" {
		request.Hooks = true
		return cp_dport
	} else if lside == "nport" {
		request.Hooks = true
		return cp_nport
	} else if lside == "policy" {
		request.Policy = true
		return cp_policy
	} else if lside == "vdom" {
		request.Policy = true
		return cp_vdom
	} else if lside == "helper" {
		request.Other = true
		return cp_helper
	} else if lside == "state" {
		request.States = true
		return cp_state
	} else if lside == "status[l]" {
		request.Basics = true
		return cp_status_l
	} else if lside == "status[r]" {
		request.Basics = true
		return cp_status_r
	} else if lside == "status" {
		request.Basics = true
		return cp_status
	} else if lside == "proto" || lside == "protocol" {
		request.Basics = true
		return cp_proto
	} else if lside == "serial" || lside == "session" {
		request.Serial = true
		return cp_serial
	} else if lside == "npuflag[o]" {
		request.Npu = true
		return cp_npuflag_o
	} else if lside == "npuflag[r]" {
		request.Npu = true
		return cp_npuflag_r
	} else if lside == "npuflag" {
		request.Npu = true
		return cp_npuflag
	} else if lside == "offload" || lside == "offloaded" {
		request.Npu = true
		return cp_offload
	} else if lside == "offload[o]" || lside == "offloaded[o]" {
		request.Npu = true
		return cp_offload_o
	} else if lside == "offload[r]" || lside == "offloaded[r]" {
		request.Npu = true
		return cp_offload_r
	} else if lside == "nturbo" {
		request.Npu = true
		return cp_nturbo
	} else if lside == "nturbo[o]" {
		request.Npu = true
		return cp_nturbo_o
	} else if lside == "nturbo[r]" {
		request.Npu = true
		return cp_nturbo_r
	} else if lside == "nooff[no]" {
		request.NpuError = true
		return cp_nooff_no
	} else if lside == "nooff[ko]" {
		request.NpuError = true
		return cp_nooff_ko
	} else if lside == "nooff[kr]" {
		request.NpuError = true
		return cp_nooff_kr
	} else if lside == "nooff[do]" {
		request.NpuError = true
		return cp_nooff_do
	} else if lside == "nooff[dr]" {
		request.NpuError = true
		return cp_nooff_dr
	} else if lside == "innpu[o]" {
		request.Npu = true
		return cp_innpu_o
	} else if lside == "innpu[f]" {
		request.Npu = true
		return cp_innpu_f
	} else if lside == "outnpu[o]" {
		request.Npu = true
		return cp_outnpu_o
	} else if lside == "outnpu[f]" {
		request.Npu = true
		return cp_outnpu_f
	} else if lside == "rate[u]" || lside == "upload" {
		request.Rate = true
		return cp_rate_u
	} else if lside == "rate[d]" || lside == "download" {
		request.Rate = true
		return cp_rate_d
	} else if lside == "rate" {
		request.Rate = true
		return cp_rate
	} else if lside == "rate[sum]" {
		request.Rate = true
		return cp_rate_sum
	} else if lside == "count[ob]" || lside == "stats[ob]" {
		request.Stats = true
		return cp_count_ob
	} else if lside == "count[op]" || lside == "stats[op]" {
		request.Stats = true
		return cp_count_op
	} else if lside == "count[oe]" || lside == "stats[oe]" {
		request.Stats = true
		return cp_count_oe
	} else if lside == "count[rb]" || lside == "stats[rb]" {
		request.Stats = true
		return cp_count_rb
	} else if lside == "count[rp]" || lside == "stats[rp]" {
		request.Stats = true
		return cp_count_rp
	} else if lside == "count[re]" || lside == "stats[re]" {
		request.Stats = true
		return cp_count_re
	} else if lside == "shapingpolicy" {
		request.Other = true
		return cp_shapingpolicy
	} else if lside == "tunnel[i]" {
		request.Other = true
		return cp_tunnel_in
	} else if lside == "tunnel[o]" {
		request.Other = true
		return cp_tunnel_out
	} else if lside == "tunnel" {
		request.Other = true
		return cp_tunnel
	} else if lside == "shaper[o]" {
		request.Shaping = true
		return cp_shaper_o
	} else if lside == "shaper[r]" {
		request.Shaping = true
		return cp_shaper_r
	} else if lside == "shaper[ip]" || lside == "shaper[pip]" {
		request.Shaping = true
		return cp_shaper_ip
	} else if lside == "shaper" {
		request.Shaping = true
		return cp_shaper
	} else if lside == "mac[i]" || lside == "mac[src]" || lside == "smac" {
		request.Macs = true
		return cp_mac_i
	} else if lside == "mac[o]" || lside == "mac[dst]" || lside == "dmac" {
		request.Macs = true
		return cp_mac_o
	} else if lside == "mac" {
		request.Macs = true
		return cp_mac
	} else if lside == "iface[oi]" || lside == "iface[io]" {
		request.Interfaces = true
		return cp_iface_in_o
	} else if lside == "iface[oo]" {
		request.Interfaces = true
		return cp_iface_out_o
	} else if lside == "iface[ri]" || lside == "iface[ir]" {
		request.Interfaces = true
		return cp_iface_in_r
	} else if lside == "iface[ro]" || lside == "iface[or]" {
		request.Interfaces = true
		return cp_iface_out_r
	} else if lside == "iface" {
		request.Interfaces = true
		return cp_iface
	} else if lside == "nexthop[o]" || lside == "nh[o]" {
		request.Interfaces = true
		return cp_nexthop_o
	} else if lside == "nexthop[r]" || lside == "nh[r]" {
		request.Interfaces = true
		return cp_nexthop_r
	} else if lside == "nexthop" || lside == "nh" {
		request.Interfaces = true
		return cp_nexthop
	} else if lside == "user" {
		request.Auth = true
		return cp_user
	} else if lside == "authserver" {
		request.Auth = true
		return cp_auth_server
	} else if lside == "authinfo" {
		request.Auth = true
		return cp_auth_info
	} else if lside == "custom" {
		request.Custom = true
		return cp_custom
	} else {
		log.Criticalf("Unknown variable \"%s\"", lside)
		os.Exit(100)
	}

	// never gets here
	return 0
}

func dumpOneLine(cond conditionOrExpression) string {
	if cond.isExpression() {
		return fmt.Sprintf("(%s %s %s !%t)", cond.(expression).lside, cond.(expression).operator, cond.(expression).rside, cond.(expression).negative)

	} else if cond.isAnd() {
		var s string
		for _, sub := range cond.(and).sub {
			s += " AND " + dumpOneLine(sub)
		}
		return "(" + s[5:] + ")"

	} else if cond.isOr() {
		var s string
		for _, sub := range cond.(or).sub {
			s += " OR " + dumpOneLine(sub)
		}
		return "(" + s[4:] + ")"
	}

	return "ERROR" // this should never happen...
}

func dumpPretty(cond conditionOrExpression, indentLevel int) string {
	var ret string

	if cond.isExpression() {
		var negative string
		if cond.(expression).negative { negative = " ! " }

		ret += strings.Repeat("\t", indentLevel)
		ret += fmt.Sprintf("EXPRESSION(%s%d \"%s\" \"%s\")", negative, cond.(expression).lside, cond.(expression).operator, cond.(expression).rside)
		ret += "\n"
		return ret

	} else if cond.isAnd() {
		s := cond.(and).sub
		/*
		if len(s) == 1 {
			return dumpPretty(s[0], indentLevel)
		}
		*/

		ret += strings.Repeat("\t", indentLevel)
		ret += "AND {\n"
		for _, sub := range s {
			ret += dumpPretty(sub, indentLevel+1)
		}
		ret += strings.Repeat("\t", indentLevel)
		ret += "}\n"
		return ret

	} else if cond.isOr() {
		s := cond.(or).sub
		/*
		if len(s) == 1 {
			return dumpPretty(s[0], indentLevel)
		}
		*/

		ret += strings.Repeat("\t", indentLevel)
		ret += "OR {\n"
		for _, sub := range s {
			ret += dumpPretty(sub, indentLevel+1)
		}
		ret += strings.Repeat("\t", indentLevel)
		ret += "}\n"
		return ret

	}

	// never get here
	return ""
}

/* 
 * Matching conditions
 */

func (c *Condition) match(cond conditionOrExpression, session *fortisession.Session) bool {
	if cond.isExpression() {
		rside := cond.(expression).rside

		// if nested formatter is used, replace rside with its result
		if cond.(expression).formatter != nil {
			rside = cond.(expression).formatter.Format(session)
			log.Debugf("formatted right side: %s", rside)
		}

		tmp := c.expression_matches(cond.(expression).lside, cond.(expression).operator, rside, cond.(expression).extra, session)
		if cond.(expression).negative { return !tmp } else { return tmp }

	} else if cond.isAnd() {
		for _, sub := range cond.(and).sub {
			if c.match(sub, session) == false { return false }
		}
		return true

	} else if cond.isOr() {
		for _, sub := range cond.(or).sub {
			if c.match(sub, session) == true { return true }
		}
		return false
	}

	return false // this should never happen...
}

func (c *Condition) expression_matches(lside conditionerParameter, operator string, rside string, extra string, session *fortisession.Session) bool {
	var result bool
	var reverse bool

	if len(operator) > 0 && operator[0] == '!' {
		reverse   = true
		operator  = operator[1:]
	}

	if lside == cp_host {
		src_ip, _, dst_ip, _, nat_ip, _, _ := session.GetPeers()
		src := c.compareIP(src_ip, operator, rside, "src")
		dst := c.compareIP(dst_ip, operator, rside, "dst")
		nat := c.compareIP(nat_ip, operator, rside, "nat")
		result = src || dst || nat

	} else if lside == cp_shost {
		src_ip, _, _, _, _, _, _ := session.GetPeers()
		result = c.compareIP(src_ip, operator, rside, "src")

	} else if lside == cp_dhost {
		_, _, dst_ip, _, _, _, _ := session.GetPeers()
		result = c.compareIP(dst_ip, operator, rside, "dst")

	} else if lside == cp_nhost {
		_, _, _, _, nat_ip, _, _ := session.GetPeers()
		result = c.compareIP(nat_ip, operator, rside, "nat")

	} else if lside == cp_port {
		_, src_port, _, dst_port, _, nat_port, _ := session.GetPeers()
		src := c.compareTextNumbers(uint64(src_port), operator, rside, "port(src)")
		dst := c.compareTextNumbers(uint64(dst_port), operator, rside, "port(dst)")
		nat := c.compareTextNumbers(uint64(nat_port), operator, rside, "port(nat)")
		result = src || dst || nat

	} else if lside == cp_sport {
		_, src_port, _, _, _, _, _ := session.GetPeers()
		result = c.compareTextNumbers(uint64(src_port), operator, rside, "sport")

	} else if lside == cp_dport {
		_, _, _, dst_port, _, _, _ := session.GetPeers()
		result = c.compareTextNumbers(uint64(dst_port), operator, rside, "dport")

	} else if lside == cp_nport {
		_, _, _, _, _, nat_port, _ := session.GetPeers()
		result = c.compareTextNumbers(uint64(nat_port), operator, rside, "nport")

	} else if lside == cp_policy {
		result = c.check_policy(session.Policy.Id, operator, rside)

	} else if lside == cp_vdom {
		result = c.compareTextNumbers(uint64(session.Policy.Vdom), operator, rside, "vdom")

	} else if lside == cp_helper {
		result = c.compareString(session.Other.Helper, operator, rside, "helper")

	} else if lside == cp_state {
		result = c.check_state(session.States, operator, rside)

	} else if lside == cp_status_l {
		result = c.check_status(session.Basics.StateL, operator, rside)

	} else if lside == cp_status_r {
		result = c.check_status(session.Basics.StateR, operator, rside)

	} else if lside == cp_status {
		left  := c.check_status(session.Basics.StateL, operator, rside)
		right := c.check_status(session.Basics.StateR, operator, rside)
		result = left || right

	} else if lside == cp_proto {
		result = c.check_proto(session.Basics.Protocol, operator, rside)

	} else if lside == cp_serial {
		result = c.compareTextNumbers(session.Serial, operator, rside, "serial")

	} else if lside == cp_npuflag {
		org := c.compareTextNumbers(uint64(session.Npu.Flag_org), operator, rside, "npuflag[o]")
		rev := c.compareTextNumbers(uint64(session.Npu.Flag_rev), operator, rside, "npuflag[r]")
		result = org && rev

	} else if lside == cp_npuflag_o {
		result = c.compareTextNumbers(uint64(session.Npu.Flag_org), operator, rside, "npuflag[o]")

	} else if lside == cp_npuflag_r {
		result = c.compareTextNumbers(uint64(session.Npu.Flag_rev), operator, rside, "npuflag[r]")

	} else if lside == cp_offload {
		org := c.compareTextNumbers(uint64(session.Npu.Offload_org), operator, rside, "offload[o]")
		rev := c.compareTextNumbers(uint64(session.Npu.Offload_rev), operator, rside, "offload[r]")
		result = org && rev

	} else if lside == cp_offload_o {
		result = c.compareTextNumbers(uint64(session.Npu.Offload_org), operator, rside, "offload[o]")

	} else if lside == cp_offload_r {
		result = c.compareTextNumbers(uint64(session.Npu.Offload_rev), operator, rside, "offload[r]")

	} else if lside == cp_nturbo {
		org := c.compareTextNumbers(uint64(session.Npu.Nturbo_org), operator, rside, "nturbo[o]")
		rev := c.compareTextNumbers(uint64(session.Npu.Nturbo_rev), operator, rside, "nturbo[r]")
		result = org && rev

	} else if lside == cp_nturbo_o {
		result = c.compareTextNumbers(uint64(session.Npu.Nturbo_org), operator, rside, "nturbo[o]")

	} else if lside == cp_nturbo_r {
		result = c.compareTextNumbers(uint64(session.Npu.Nturbo_rev), operator, rside, "nturbo[r]")

	} else if lside == cp_nooff_no {
		result = c.compareString(session.NpuError.NoOffloadReason, operator, rside, "nooff[no]")

	} else if lside == cp_nooff_ko {
		result = c.compareString(session.NpuError.Kernel_org, operator, rside, "nooff[ko]")

	} else if lside == cp_nooff_kr {
		result = c.compareString(session.NpuError.Kernel_rev, operator, rside, "nooff[kr]")

	} else if lside == cp_nooff_do {
		result = c.compareString(session.NpuError.Driver_org, operator, rside, "nooff[do]")

	} else if lside == cp_nooff_dr {
		result = c.compareString(session.NpuError.Driver_rev, operator, rside, "nooff[dr]")

	} else if lside == cp_innpu_o {
		result = c.compareTextNumbers(uint64(session.Npu.InNpu_org), operator, rside, "innpu[o]")

	} else if lside == cp_innpu_f {
		result = c.compareTextNumbers(uint64(session.Npu.InNpu_fwd), operator, rside, "innpu[f]")

	} else if lside == cp_outnpu_o {
		result = c.compareTextNumbers(uint64(session.Npu.OutNpu_org), operator, rside, "outnpu[o]")

	} else if lside == cp_outnpu_f {
		result = c.compareTextNumbers(uint64(session.Npu.OutNpu_fwd), operator, rside, "outnpu[f]")

	} else if lside == cp_rate_u {
		result = check_rate(session.Rate.Tx_Bps, operator, rside)

	} else if lside == cp_rate_d {
		result = check_rate(session.Rate.Rx_Bps, operator, rside)

	} else if lside == cp_rate {
		upload   := check_rate(session.Rate.Tx_Bps, operator, rside)
		download := check_rate(session.Rate.Rx_Bps, operator, rside)
		result = upload || download

	} else if lside == cp_rate_sum {
		result = check_rate(session.Rate.Rx_Bps+session.Rate.Tx_Bps, operator, rside)

	} else if lside == cp_count_ob {
		if !session.Stats.Valid_org { result = false
		} else if c.compareTextNumbers(session.Stats.Bytes_org, operator, rside, "count[ob]") { result = true
		} else { result = false }

	} else if lside == cp_count_op {
		if !session.Stats.Valid_org { result = false
		} else if c.compareTextNumbers(session.Stats.Packets_org, operator, rside, "count[op]") { result = true
		} else { result = false }

	} else if lside == cp_count_oe {
		if !session.Stats.Valid_org { result = false
		} else if c.compareTextNumbers(session.Stats.Errors_org, operator, rside, "count[oe]") { result = true
		} else { result = false }

	} else if lside == cp_count_rb {
		if !session.Stats.Valid_rev { result = false
		} else if c.compareTextNumbers(session.Stats.Bytes_rev, operator, rside, "count[rb]") { result = true
		} else { result = false }

	} else if lside == cp_count_rp {
		if !session.Stats.Valid_rev { result = false
		} else if c.compareTextNumbers(session.Stats.Packets_rev, operator, rside, "count[rp]") { result = true
		} else { result = false }

	} else if lside == cp_count_re {
		if !session.Stats.Valid_rev { result = false
		} else if c.compareTextNumbers(session.Stats.Errors_rev, operator, rside, "count[re]") { result = true
		} else { result = false }

	} else if lside == cp_shapingpolicy {
		result = c.compareTextNumbers(uint64(session.Other.ShapingPolicyId), operator, rside, "shapingpolicy")

	} else if lside == cp_tunnel_in {
		result = c.compareString(session.Other.Tunnel_in, operator, rside, "tunnel[i]")

	} else if lside == cp_tunnel_out {
		result = c.compareString(session.Other.Tunnel_out, operator, rside, "tunnel[o]")

	} else if lside == cp_tunnel {
		in  := c.compareString(session.Other.Tunnel_in, operator, rside, "tunnel[i]")
		out := c.compareString(session.Other.Tunnel_out, operator, rside, "tunnel[o]")
		result = in || out

	} else if lside == cp_shaper_o {
		result = c.compareString(session.Shaping.Shaper_org, operator, rside, "shaper[o]")

	} else if lside == cp_shaper_r {
		result = c.compareString(session.Shaping.Shaper_rev, operator, rside, "shaper[r]")

	} else if lside == cp_shaper_ip {
		result = c.compareString(session.Shaping.Shaper_ip, operator, rside, "shaper[ip]")

	} else if lside == cp_shaper {
		org := c.compareString(session.Shaping.Shaper_org, operator, rside, "shaper[o]")
		rev := c.compareString(session.Shaping.Shaper_rev, operator, rside, "shaper[r]")
		pip := c.compareString(session.Shaping.Shaper_ip, operator, rside, "shaper[ip]")
		result = org || rev || pip

	} else if lside == cp_mac_i {
		result = c.compareString(session.Macs.Src, operator, rside, "mac[i]")

	} else if lside == cp_mac_o {
		result = c.compareString(session.Macs.Dst, operator, rside, "mac[o]")

	} else if lside == cp_mac {
		src := c.compareString(session.Macs.Src, operator, rside, "mac[i]")
		dst := c.compareString(session.Macs.Dst, operator, rside, "mac[o]")
		result = src || dst

	} else if lside == cp_iface_in_o {
		result = c.compareTextNumbers(uint64(session.Interfaces.In_org), operator, rside, "iface[oi]")

	} else if lside == cp_iface_out_o {
		result = c.compareTextNumbers(uint64(session.Interfaces.Out_org), operator, rside, "iface[oo]")

	} else if lside == cp_iface_in_r {
		result = c.compareTextNumbers(uint64(session.Interfaces.In_rev), operator, rside, "iface[ri]")

	} else if lside == cp_iface_out_r {
		result = c.compareTextNumbers(uint64(session.Interfaces.Out_rev), operator, rside, "iface[ro]")

	} else if lside == cp_iface {
		i_oi := c.compareTextNumbers(uint64(session.Interfaces.In_org), operator, rside, "iface[oi]")
		i_oo := c.compareTextNumbers(uint64(session.Interfaces.Out_org), operator, rside, "iface[oo]")
		i_ri := c.compareTextNumbers(uint64(session.Interfaces.In_rev), operator, rside, "iface[ri]")
		i_ro := c.compareTextNumbers(uint64(session.Interfaces.Out_rev), operator, rside, "iface[ro]")
		result = i_oi || i_oo || i_ri || i_ro

	} else if lside == cp_nexthop_o {
		result = c.compareIP(session.Interfaces.NextHop_org, operator, rside, "nexthop[o]")

	} else if lside == cp_nexthop_r {
		result = c.compareIP(session.Interfaces.NextHop_rev, operator, rside, "nexthop[r]")

	} else if lside == cp_nexthop {
		org := c.compareIP(session.Interfaces.NextHop_org, operator, rside, "nexthop[o]")
		rev := c.compareIP(session.Interfaces.NextHop_rev, operator, rside, "nexthop[r]")
		result = org || rev

	} else if lside == cp_user {
		result = c.compareString(session.Auth.User, operator, rside, "user")

	} else if lside == cp_auth_server {
		result = c.compareString(session.Auth.Profile, operator, rside, "authserver")

	} else if lside == cp_auth_info {
		result = c.compareTextNumbers(uint64(session.Auth.AuthInfo), operator, rside, "authinfo")

	} else if lside == cp_custom {
		v, exists := session.Custom[extra]
		if !exists {
			log.Criticalf("Nonexisting custom variable \"%s\"", extra)
			os.Exit(100)
		}
		result = c.check_custom(v, operator, rside)

	} else {
		log.Criticalf("Unknown filter: \"%s\" \"%s\" \"%s\"", lside, operator, rside)
		os.Exit(100)
	}


	log.Tracef("Check session 0x%x: \"%s\" \"%s\" \"%s\" -> %t\n", session.Serial, lside, operator, rside, result)
	if !reverse {
		return result
	} else {
		return !result
	}
}

func (c *Condition) check_policy(session_policy uint32, operator string, rside string) bool {
	if rside == "internal" {
		rside = "4294967295"
	}

	return c.compareTextNumbers(uint64(session_policy), operator, rside, "policy")
}

func (c *Condition) check_state(session_states []fortisession.State, operator string, rside string) bool {
	var ret bool

	for _, s := range session_states {
		if c.compareString(string(s), operator, rside, "state") {
			ret = true
			break
		}
	}

	return ret
}

func (c *Condition) check_status(session_status uint8, operator string, rside string) bool {
	rside = strings.ToLower(rside)
	if len(rside) == 0                                { rside = "1"
	} else if strings.HasPrefix(rside, "n")           { rside = "0"
	} else if strings.HasPrefix(rside, "e")           { rside = "1"
	} else if strings.HasPrefix(rside, "syn-s")       { rside = "2"
	} else if strings.HasPrefix(rside, "syn_s")       { rside = "2"
	} else if strings.HasPrefix(rside, "ss")          { rside = "2"
	} else if strings.HasPrefix(rside, "syn-r")       { rside = "3"
	} else if strings.HasPrefix(rside, "syn_r")       { rside = "3"
	} else if strings.HasPrefix(rside, "sr")          { rside = "3"
	} else if strings.HasPrefix(rside, "fin-wait1")   { rside = "4"
	} else if strings.HasPrefix(rside, "fin_wait1")   { rside = "4"
	} else if strings.HasPrefix(rside, "fw1")         { rside = "4"
	} else if strings.HasPrefix(rside, "fin-wait2")   { rside = "5"
	} else if strings.HasPrefix(rside, "fin_wait2")   { rside = "5"
	} else if strings.HasPrefix(rside, "fw2")         { rside = "5"
	} else if strings.HasPrefix(rside, "t")           { rside = "6"
	} else if rside == "close"                        { rside = "7"
	} else if strings.HasPrefix(rside, "close-")      { rside = "8"
	} else if strings.HasPrefix(rside, "close_")      { rside = "8"
	} else if strings.HasPrefix(rside, "cw")          { rside = "8"
	} else if strings.HasPrefix(rside, "la")          { rside = "9"
	} else if strings.HasPrefix(rside, "li")          { rside = "10"
	} else if strings.HasPrefix(rside, "closi")       { rside = "11"
	} else if rside == "seen"                         { rside = "1"
	} else if rside == "unseen"                       { rside = "0"
	}

	return c.compareTextNumbers(uint64(session_status), operator, rside, "status")
}

func (c *Condition) check_proto(session_proto uint16, operator string, rside string) bool {
	rside = strings.ToLower(rside)
	if rside == "icmp"          { rside = "1"
	} else if rside == "tcp"    { rside = "6"
	} else if rside == "udp"    { rside = "17"
	} else if rside == "ipv6"   { rside = "41"
	} else if rside == "gre"    { rside = "47"
	} else if rside == "esp"    { rside = "50"
	}

	return c.compareTextNumbers(uint64(session_proto), operator, rside, "proto")
}

func check_rate(rate_Bps uint64, operator string, rside string) bool {
	var find_rate_b uint64
	var find_unit string = "Bps"

	for i, part := range strings.Split(rside, " ") {
		if i == 0 {
			tmp, err := strconv.ParseUint(part, 10, 64)
			if err != nil {
				log.Criticalf("Check rate error: unable to parse number \"%s\"", part)
				os.Exit(100)
			}
			find_rate_b = tmp

		} else {
			if len(part) > 0 {
				find_unit = strings.TrimSpace(part)
				break
			}
		}
	}

	// units format check
	if find_unit[len(find_unit)-2:] != "ps" {
		log.Criticalf("Check rate error: unknown units (1) \"%s\"", find_unit)
		os.Exit(100)
	} else {
		find_unit = find_unit[:len(find_unit)-2]
	}

	// bit vs byte
	if find_unit[len(find_unit)-1] == 'B' {
		find_rate_b *= 8
		find_unit = find_unit[:len(find_unit)-1]
	} else if find_unit[len(find_unit)-1] == 'b' {
		find_unit = find_unit[:len(find_unit)-1]
	} else {
		log.Criticalf("Check rate error: unknown units (2) \"%s\"", find_unit)
		os.Exit(100)
	}

	// multiply
	find_unit = strings.ToLower(find_unit)
	if find_unit == "k" { find_rate_b *= 1000
	} else if find_unit == "ki" { find_rate_b *= 1024
	} else if find_unit == "m" { find_rate_b *= 1000*1000
	} else if find_unit == "mi" { find_rate_b *= 1024*1024
	} else if find_unit == "g" { find_rate_b *= 1000*1000*1000
	} else if find_unit == "gi" { find_rate_b *= 1024*1024*1024
	} else if find_unit == "t" { find_rate_b *= 1000*1000*1000*1000
	} else if find_unit == "ti" { find_rate_b *= 1024*1024*1024*1024
	} else {
		log.Criticalf("Check rate error: unknown units (3) \"%s\"", find_unit)
		os.Exit(100)
	}

	// test
	result, err := compareUInt64(rate_Bps, operator, find_rate_b/8)
	if err != nil {
		log.Criticalf("Check rate error: %s", err)
		os.Exit(100)
	}

	return result
}

func (c *Condition) check_custom(custom *multivalue.MultiValue, operator string, rside string) bool {
	if custom.IsString() {
		return c.compareString(custom.GetString(), operator, rside, "custom")
	} else if custom.IsUint64() {
		return c.compareTextNumbers(custom.GetUint64(), operator, rside, "custom")
	} else if custom.IsFloat64() {
		log.Warningf("To compare float type custom variable, it is first converted to unsigned integer")
		return c.compareTextNumbers(uint64(custom.GetUint64()), operator, rside, "custom")
	} else if custom.IsEmpty() {
		return false
	} else {
		log.Criticalf("Unknown custom field type")
		os.Exit(100)
	}

	// never gets here
	return false
}

/* 
 * auxiliary functions for matching
 */

func compareUInt64(left uint64, operator string, right uint64) (bool, error) {
	if operator == "" || operator == "==" || operator == "=" || operator == "eq" || operator == "is" {
		if left == right { return true, nil} else { return false, nil }

	} else if operator == "<>" || operator == "ne" || operator == "not" || operator == "!=" {
		if left != right { return true, nil} else { return false, nil }

	} else if operator == ">" || operator == "gt" {
		if left > right { return true, nil} else { return false, nil }

	} else if operator == ">=" || operator == "ge" {
		if left >= right { return true, nil} else { return false, nil }

	} else if operator == "<" || operator == "lt" {
		if left < right { return true, nil} else { return false, nil }

	} else if operator == "<=" || operator == "le" {
		if left <= right { return true, nil} else { return false, nil }

	} else {
		return false, fmt.Errorf("unknown integer operator \"%s\"", operator)
	}
}

func (c *Condition) compareString(left string, operator string, right string, logtext string) (bool) {
	if len(operator) == 0 && len(right) == 0 {
		if len(left) == 0 { return false } else { return true }
	}

	if len(operator) > 0 && operator[0] == '#' {
		left     = strings.ToLower(left)
		right    = strings.ToLower(right)
		operator = operator[1:]
	}

	if operator == "" || operator == "==" || operator == "=" || operator == "eq" || operator == "is" {
		return left == right

	} else if operator == "not" || operator == "!=" || operator == "<>" || operator == "ne" {
		return left != right

	} else if operator == "prefix" || operator == "starts" || operator == "start" {
		return strings.HasPrefix(left, right)

	} else if operator == "suffix" || operator == "ends" || operator == "end" {
		return strings.HasSuffix(left, right)

	} else if operator == "contains" || operator == "contain" || operator == "c" || operator == "has" {
		return strings.Contains(left, right)

	} else {
		log.Criticalf("Generic check \"%s\" unknown string operator \"%s\"", logtext, operator)
		os.Exit(100)
	}

	// never gets here
	return false
}

func (c *Condition) compareTextNumbers(left uint64, operator string, rside string, logtext string) bool {
	if len(operator) == 0 && len(rside) == 0 {
		if left > 0 { return true } else { return false }
	}

	rside = strings.ToLower(rside)
	var find uint64

	base := 10
	if strings.HasPrefix(rside, "0x") {
		base = 16
		rside = rside[2:]
	}

	num, err := strconv.ParseUint(rside, base, 64)
	if err != nil {
		log.Criticalf("Generic check \"%s\" error: %s", logtext, err)
		os.Exit(100)
	}

	find = uint64(num)

	result, err := compareUInt64(uint64(left), operator, find)
	if err != nil {
		log.Criticalf("Generic check \"%s\" serial error: %s", logtext, err)
		os.Exit(100)
	}

	return result
}

func (c *Condition) compareIP(session_ip net.IP, operator string, rside string, logtext string) bool {

	if operator == "" || operator == "=" || operator == "==" || operator == "is" {
		ip := net.ParseIP(rside)
		if ip == nil {
			log.Criticalf("Check IP \"%s\" error: IP not parsable \"%s\"", logtext, rside)
			os.Exit(100)
		}
		return ip.Equal(session_ip)

	} else if operator == "!=" || operator == "<>" || operator == "not" {
		ip := net.ParseIP(rside)
		if ip == nil {
			log.Criticalf("Check IP \"%s\" error: IP not parsable \"%s\"", logtext, rside)
			os.Exit(100)
		}
		return !ip.Equal(session_ip)

	} else if operator == "in" {
		_, net, err := net.ParseCIDR(rside)
		if err != nil {
			log.Criticalf("Check IP \"%s\" error: IP CIDR not parsable \"%s\"", logtext, rside)
			os.Exit(100)
		}
		return net.Contains(session_ip)

	} else {
		log.Criticalf("Check IP \"%s\" error: unknown ip operator \"%s\"\n", logtext, operator)
		os.Exit(100)
	}

	return false // this should never happen
}
