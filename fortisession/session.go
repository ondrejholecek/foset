// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

// Parse one plain text session collected on FortiGate device
// using `diagnose sys session list` command.
//
// Currently only IPv4 sessions are supported.
package fortisession

import (
	"strconv"
	"unicode"
	"bytes"
	"strings"
	"net"
	"github.com/juju/loggo"
	"foset/fortisession/multivalue"
)

var log loggo.Logger

// InitLog initializes the logging system with the object passed from the calling program. 
// It is expected that this object is already initialized with loggo.GetLogger()
// function or something similar.
func InitLog(l loggo.Logger) { log = l }

// IpPort binds the IP address and port number.
type IpPort struct {
	Ip   net.IP
	Port uint16
}

// Hook contains one `hook` with Hook "time", direction and action. 
// And source, destination, and natted IP and port.
type Hook struct {
	Hook string
	Dir string
	Act string
	Src IpPort
	Dst IpPort
	Nat IpPort
}

// State is alias for string.
type State string

// Basics contains very basic information about the session.
type Basics struct {
	Protocol  uint16
	StateL    uint8
	StateR    uint8
	Duration  uint64
	Expire    uint64
	Timeout   uint64
}

// Stats contains the statistics of the session, such as
// the number of bytes, packets and errors in each direction.
//
// There is also Valid_org and and Valid_rev booleans to signalize
// the validity of the counters. 
//
type Stats struct {
	Bytes_org   uint64
	Packets_org uint64
	Errors_org  uint64
	Valid_org   bool
	Bytes_rev   uint64
	Packets_rev uint64
	Errors_rev  uint64
	Valid_rev   bool
}

// Other contains the fields not fitting anywhere else.
type Other struct {
	HAid              uint8
	Helper            string
	ShapingPolicyId   uint32
	Tunnel_in         string
	Tunnel_out        string
}

// Shaping related information
type Shaping struct {
	Shaper_org    string
	Shaper_rev    string
	Shaper_ip     string
}

// Macs contains source and destination MAC addresses
type Macs struct {
	Src   string
	Dst   string
}

// Interface contains interface pairs in both directions and
// next hops IP addresses
type Interfaces struct {
	In_org       uint32
	Out_org      uint32
	In_rev       uint32
	Out_rev      uint32
	NextHop_org  net.IP
	NextHop_rev  net.IP
}

// Rate contains the speed by bytes per seconds for each direction.
// "TX" speed is upload in the original direction of the session (from client to server)
// and "RX" speed is the reverse direction (from server to client).
type Rate struct {
	Tx_Bps  uint64
	Rx_Bps  uint64
}

// Auth contains information about authenticated user.
type Auth struct {
	User     string
	Profile  string
	AuthInfo uint64
}

// Npu contains NPU related fields such as HW offloading or nTurbo.
// Each value is array where the first number is original direction
// and the second is reverse.
//
type Npu struct {
	Offload_org       uint8
	Offload_rev       uint8
	Nturbo_org        uint8
	Nturbo_rev        uint8
	InNpu_org         uint8
	InNpu_fwd         uint8
	InNpu_org_valid   bool
	InNpu_fwd_valid   bool
	OutNpu_org        uint8
	OutNpu_fwd        uint8
	OutNpu_org_valid  bool
	OutNpu_fwd_valid  bool
	Flag_org          uint8
	Flag_rev          uint8
}

// NpuError contains the no offload reasons.
// NoOffloadReason is simple string extracted from `no_ofld_reason` field.
// The other fields are extracted from `ofld_fail_reason` field and split.
//
type NpuError struct {
	NoOffloadReason string
	Kernel_org      string
	Kernel_rev      string
	Driver_org      string
	Driver_rev      string
}

// Policy contains the VDOM id and the policy id within the VDOM.
type Policy struct {
	Id      uint32
	Vdom    uint32
}

// Session is the main structure containing all the parsed information.
//
// Based on the values enabled in `SessionDataRequest` structure passed to `Parse` function,
// some values can be empty and may contain the type's default value instead of the actual
// values from the plain-text session.
type Session struct {
	Plain      string
	Serial     uint64
	Hooks      []Hook
	States     []State
	Basics     *Basics
	Stats      *Stats
	Rate       *Rate
	Npu        *Npu
	Policy     *Policy
	Other      *Other
	NpuError   *NpuError
	Shaping    *Shaping
	Macs       *Macs
	Interfaces *Interfaces
	Auth       *Auth
	// aux
	Custom     map[string]*multivalue.MultiValue
}

// SessionDataRequest specifies which fields should be extracted by the `Parse` function.
// Extracting only fields the application is actually interested in speeds up 
// the processing significantly.
type SessionDataRequest struct {
	Plain      bool
	Serial     bool
	Hooks      bool
	States     bool
	Basics     bool
	Stats      bool
	Rate       bool
	Npu        bool
	Policy     bool
	Other      bool
	NpuError   bool
	Shaping    bool
	Macs       bool
	Interfaces bool
	Auth       bool
	// aux
	Custom     bool
}

// SetAll enables parsing of all possible fields
// from the session plain text.
func (req *SessionDataRequest) SetAll() {
	req.Plain      = true
	req.Serial     = true
	req.Hooks      = true
	req.States     = true
	req.Basics     = true
	req.Stats      = true
	req.Rate       = true
	req.Npu        = true
	req.Policy     = true
	req.Other      = true
	req.NpuError   = true
	req.Shaping    = true
	req.Macs       = true
	req.Interfaces = true
	req.Auth       = true
	// aux
	req.Custom     = true
}


// GetPeers returns source, destination IP address and ports for the current session.
//
// This information is extracted from the Hooks structure.
func (session *Session) GetPeers() (src_ip net.IP, src_port uint16, dst_ip net.IP, dst_port uint16, nat_ip net.IP, nat_port uint16, ok bool) {
	ok = false
	for _, hook := range session.Hooks {
		if hook.Dir == "org" {
			src_ip   = hook.Src.Ip
			src_port = hook.Src.Port
			dst_ip   = hook.Dst.Ip
			dst_port = hook.Dst.Port
			nat_ip   = hook.Nat.Ip
			nat_port = hook.Nat.Port
			ok       = true
			break
		}
	}

	return
}

// Parse takes one plain-text session as byte array and returns
// one Session structure that contains the information extracted from it.
//
// Parameter `requested` specify which data the caller is interested in.
// For fields that are set to `false` Parse will not even try to extract
// the values.
func Parse(data []byte, requested *SessionDataRequest) *Session {
	var s Session
	log.Tracef("Parsing session:\n%s\n---end---\n", string(data))

	// add the final new line to be easily able to match lines
	if data[len(data)-1] != '\n' {
		data = append(data, '\n')
	}

	if requested.Plain      { s.Plain      = string(data[1:])       }
	if requested.Serial     { s.Serial     = get_serial(&data)      }
	if requested.States     { s.States     = get_states(&data)      }
	if requested.Hooks      { s.Hooks      = get_hooks(&data)       }
	if requested.Basics     { s.Basics     = get_basics(&data)      }
	if requested.Stats      { s.Stats      = get_stats(&data)       }
	if requested.Rate       { s.Rate       = get_rate(&data)        }
	if requested.Npu        { s.Npu        = get_npu(&data)         }
	if requested.Policy     { s.Policy     = get_policy(&data)      }
	if requested.Other      { s.Other      = get_other(&data)       }
	if requested.NpuError   { s.NpuError   = get_npu_error(&data)   }
	if requested.Shaping    { s.Shaping    = get_shaping(&data)     }
	if requested.Macs       { s.Macs       = get_macs(&data)        }
	if requested.Interfaces { s.Interfaces = get_interfaces(&data)  }
	if requested.Auth       { s.Auth       = get_auth(&data)        }
	// aux
	if requested.Custom     { s.Custom = make(map[string]*multivalue.MultiValue)    }

	return &s
}

func get_serial(data *[]byte) uint64 {
	lines := extract_lines(data, []byte("serial="))
	if len(lines) == 0 { return 0 }

	var k, v string
	var ok bool

	for _, line := range lines {
		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "serial" {
				val, _ := strconv.ParseUint(v, 16, 64)
				return val
			}
		}
	}

	return 0
}

func get_other(data *[]byte) *Other {
	var other Other
	var k, v string
	var ok bool

	for _, line := range find_lines_with_field(data, []byte("ha_id"), nil) {
		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "ha_id" {
				tmp, _ := strconv.ParseUint(v, 10, 8)
				other.HAid = uint8(tmp)

			} else if k == "helper" {
				other.Helper = v

			} else if k == "shaping_policy_id" {
				tmp, _ := strconv.ParseUint(v, 10, 32)
				other.ShapingPolicyId = uint32(tmp)

			} else if k == "tunnel" {
				tmp := strings.Split(v, "/")
				if len(tmp) != 2 {
					log.Debugf("Cannot parse tunnel in \"%s\"", line)
					continue
				}
				other.Tunnel_in = tmp[1]
				other.Tunnel_out = tmp[0]
			}
		}
	}

	return &other
}

func get_states(data *[]byte) []State {
	states  := make([]State, 0)
	offsets := make([]int, 0)

	for i, line := range find_lines_with_field(data, []byte("state"), &offsets) {
		partial := line[offsets[i]:]
		k, v, ok := extract_pair(&partial, []byte("="), []byte(", "))
		if !ok { break }
		if k != "state" { continue }

		for _, state := range strings.Split(v, " ") {
			if len(state) == 0 { continue }
			states = append(states, State(state))
		}
	}

	return states
}

func get_auth(data *[]byte) *Auth {
	var auth    Auth
	var lines   [][]byte

	lines = append(lines, find_lines_with_field(data, []byte("user"), nil)...)
	lines = append(lines, find_lines_with_field(data, []byte("auth_info"), nil)...)

	for _, line := range lines {
		for {
			k, v, ok := extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "user" {
				auth.User = v
			} else if k == "auth_server" {
				auth.Profile = v
			} else if k == "auth_info" {
				num, err := strconv.ParseUint(v, 10, 64)
				if err != nil {
					log.Warningf("Unable to parse \"auth_info\" data \"%s\": %s", v, err)
				}
				auth.AuthInfo = uint64(num)
			}
		}
	}

	return &auth
}

func get_npu(data *[]byte) *Npu {
	var npu Npu

	var k, v string
	var ok bool

	lines := extract_lines(data, []byte("npu info:"))
	for _, line := range lines {
		line = line[10:]

		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(", "))
			if !ok { break }

			if k == "offload" {
				tmp := slash_numbers(v)
				npu.Offload_org = uint8(tmp[0])
				npu.Offload_rev = uint8(tmp[1])
			} else if k == "ips_offload" {
				tmp := slash_numbers(v)
				npu.Nturbo_org = uint8(tmp[0])
				npu.Nturbo_rev = uint8(tmp[1])
			} else if k == "flag" {
				tmp := slash_numbers(v)
				npu.Flag_org = uint8(tmp[0])
				npu.Flag_rev = uint8(tmp[1])
			}
		}
	}

	for _, line := range find_lines_with_field(data, []byte("in_npu"), nil) {
		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "in_npu" {
				tmp := slash_numbers(v)
				if tmp[0] == 0 {
					npu.InNpu_org_valid = false
				} else {
					npu.InNpu_org  = uint8(tmp[0]-1)
					npu.InNpu_org_valid = true
				}
				if tmp[1] == 0 {
					npu.InNpu_fwd_valid = false
				} else {
					npu.InNpu_fwd  = uint8(tmp[1]-1)
					npu.InNpu_fwd_valid = true
				}
			} else if k == "out_npu" {
				tmp := slash_numbers(v)
				if tmp[0] == 0 {
					npu.OutNpu_org_valid = false
				} else {
					npu.OutNpu_org  = uint8(tmp[0]-1)
					npu.OutNpu_org_valid = true
				}
				if tmp[1] == 0 {
					npu.OutNpu_fwd_valid = false
				} else {
					npu.OutNpu_fwd  = uint8(tmp[1]-1)
					npu.OutNpu_fwd_valid = true
				}
			}
		}
	}

	return &npu
}

func get_npu_error(data *[]byte) *NpuError {
	var npue NpuError

	lines := extract_lines(data, []byte("no_ofld_reason:"))
	for _, line := range lines {
		npue.NoOffloadReason = string(bytes.TrimSpace(line[15:]))
	}

	lines = extract_lines(data, []byte("ofld_fail_reason(kernel, drv):"))
	for _, line := range lines {
		tmp := strings.Split(string(bytes.TrimSpace(line[30:])), ", ")
		if len(tmp) != 2 { continue }

		for i, part := range tmp {
			fr := strings.Split(part, "/")
			if len(fr) != 2 { continue }

			if i == 0 {
				npue.Kernel_org = fr[0]
				npue.Kernel_rev = fr[1]

			} else if i == 1 {
				npue.Driver_org = fr[0]
				npue.Driver_rev = fr[1]
			}
		}
	}

	return &npue
}

func get_hooks(data *[]byte) []Hook {
	hooks := make([]Hook, 0)

	lines := extract_lines(data, []byte("hook="))
	for _, line := range lines {
		var hook Hook
		var k, v string
		var ok bool

		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "hook" { hook.Hook = v }
			if k == "dir"  { hook.Dir  = v }
			if k == "act"  { hook.Act  = v }
		}

		var port uint64

		k, v, ok = extract_pair(&line, []byte(":"), []byte("->"))
		port, _ = strconv.ParseUint(v, 10, 16)
		hook.Src = IpPort { Ip : net.ParseIP(k), Port : uint16(port) }

		k, v, ok = extract_pair(&line, []byte(":"), []byte("("))
		port, _ = strconv.ParseUint(v, 10, 16)
		hook.Dst = IpPort { Ip : net.ParseIP(k), Port : uint16(port) }

		k, v, ok = extract_pair(&line, []byte(":"), []byte(")"))
		port, _ = strconv.ParseUint(v, 10, 16)
		hook.Nat = IpPort { Ip : net.ParseIP(k), Port : uint16(port) }

		hooks = append(hooks, hook)
	}

	return hooks
}

func get_basics(data *[]byte) (*Basics) {
	var basics Basics

	lines := extract_lines(data, []byte("session info: "))
	for _, line := range lines {
		line = line[len("session info: "):]

		var k, v string
		var ok bool

		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "proto" {
				tmp, _ := strconv.ParseUint(v, 10, 16)
				basics.Protocol = uint16(tmp)
			} else if k == "proto_state" {
				tmp1, _ := strconv.ParseUint(string(v[0]), 16, 8)
				tmp2, _ := strconv.ParseUint(string(v[1]), 16, 8)
				basics.StateL = uint8(tmp1)
				basics.StateR = uint8(tmp2)
			} else if k == "duration" {
				tmp, _ := strconv.ParseUint(v, 10, 64)
				basics.Duration = uint64(tmp)
			} else if k == "expire" {
				tmp, _ := strconv.ParseUint(v, 10, 64)
				basics.Expire = uint64(tmp)
			} else if k == "timeout" {
				tmp, _ := strconv.ParseUint(v, 10, 64)
				basics.Timeout = uint64(tmp)
			}
		}
	}

	return &basics
}

func get_rate(data *[]byte) (*Rate) {
	var rate Rate

	lines := extract_lines(data, []byte("tx speed(Bps/kbps):"))
	for _, line := range lines {
		var k, v string
		var ok bool

		for {
			k, v, ok = extract_pair(&line, []byte(": "), []byte(" "))
			if !ok { break }

			tmp1     := []byte(v)
			B, _, _  := extract_pair(&tmp1, []byte("/"), []byte(" "))
			tmp2, _  := strconv.ParseUint(B, 10, 64)

			if k == "tx speed(Bps/kbps)" {
				rate.Tx_Bps = tmp2
			} else if k == "rx speed(Bps/kbps)" {
				rate.Rx_Bps = tmp2
			}
		}
	}

	return &rate
}

func get_stats(data *[]byte) (*Stats) {
	var stats Stats

	lines := extract_lines(data, []byte("statistic(bytes/packets/allow_err): "))
	for _, line := range lines {
		line = line[len("statistic(bytes/packets/allow_err): "):]

		for {
			k, v, ok := extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }
			nums := slash_numbers(v)
			if len(nums) != 3 { continue }

			if k == "org" {
				stats.Bytes_org   = nums[0]
				stats.Packets_org = nums[1]
				stats.Errors_org  = nums[2]
				stats.Valid_org   = true

			} else if k == "reply" {
				stats.Bytes_rev   = nums[0]
				stats.Packets_rev = nums[1]
				stats.Errors_rev  = nums[2]
				stats.Valid_rev   = true

			} else {
				continue
			}
		}
	}

	return &stats
}

func get_policy(data *[]byte) *Policy {
	var policy Policy
	var k, v string
	var ok bool

	for _, line := range find_lines_with_field(data, []byte("policy_id"), nil) {
		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "policy_id" {
				tmp, _ := strconv.ParseUint(v, 10, 64)
				policy.Id = uint32(tmp)
			} else if k == "vd" {
				tmp, _ := strconv.ParseUint(v, 10, 64)
				policy.Vdom = uint32(tmp)
			}
		}
	}

	return &policy
}

func get_shaping(data *[]byte) *Shaping {
	var shaping Shaping
	var k, v string
	var ok bool

	var lines [][]byte
	lines = append(lines, find_lines_with_field(data, []byte("origin-shaper"), nil)...)
	lines = append(lines, find_lines_with_field(data, []byte("reply-shaper"), nil)...)
	lines = append(lines, find_lines_with_field(data, []byte("per_ip_shaper"), nil)...)

	for _, line := range lines {
		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "origin-shaper" {
				shaping.Shaper_org = v
			} else if k == "reply-shaper" {
				shaping.Shaper_rev = v
			} else if k == "per_ip_shaper" {
				shaping.Shaper_ip  = v
			}
		}
	}

	return &shaping
}

func get_macs(data *[]byte) *Macs{
	var macs Macs
	var k, v string
	var ok bool

	var lines [][]byte
	lines = append(lines, find_lines_with_field(data, []byte("src_mac"), nil)...)
	lines = append(lines, find_lines_with_field(data, []byte("dst_mac"), nil)...)

	for _, line := range lines {
		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "src_mac" {
				macs.Src = v
			} else if k == "dst_mac" {
				macs.Dst = v
			}
		}
	}

	return &macs
}

func get_interfaces(data *[]byte) *Interfaces {
	var ifaces Interfaces
	var k, v string
	var ok bool

	for _, line := range find_lines_with_field(data, []byte("dev"), nil) {
		d := bytes.Index(line, []byte("dev="))
		line = line[d:]

		for {
			k, v, ok = extract_pair(&line, []byte("="), []byte(" "))
			if !ok { break }

			if k == "dev" {
				for i, or := range strings.Split(v, "/") {
					for ii, io := range strings.Split(or, "->") {
						num, err := strconv.ParseUint(io, 10, 32)
						if err != nil {
							log.Warningf("Cannot extract numbers from dev field \"%s\": %s", v, err)
							break
						}

						if i == 0 && ii == 0        { ifaces.In_org   = uint32(num)
						} else if i == 0 && ii == 1 { ifaces.Out_org  = uint32(num)
						} else if i == 1 && ii == 0 { ifaces.In_rev   = uint32(num)
						} else if i == 1 && ii == 1 { ifaces.Out_rev  = uint32(num)
						} else {
							log.Warningf("Invalid format of dev field \"%s\"", v)
							break
						}
					}
				}
			} else if k == "gwy" {
				for i, nh := range strings.Split(v, "/") {
					if i == 0  { ifaces.NextHop_org = net.ParseIP(nh)
					} else if i == 1 { ifaces.NextHop_rev = net.ParseIP(nh)
					} else {
						log.Warningf("Invalid format of gwy field \"%s\"", v)
						break
					}
				}
			}
		}
	}

	return &ifaces
}

/*
 * Helper functions
 */

func extract_lines(data *[]byte, prefix []byte) [][]byte {
	to_delete := make([][]int, 0)
	lines     := make([][]byte, 0)

	offset := 0
	for {
		start := bytes.Index((*data)[offset:], append([]byte("\n"), prefix...))
		if start == -1 { break }
		end   := bytes.Index((*data)[offset+start+1:], []byte("\n"))
		if end   == -1 { break }

//		log.Tracef("Extract lines: start: %d, end: %d, data: [%s]\n", offset+start, offset+start+1+end, (*data)[offset+start+1 : offset+start+1+end])
		lines = append(lines, (*data)[offset+start+1 : offset+start+1+end])
		to_delete = append(to_delete, []int{offset+start+1, offset+start+1+end+1})
		offset = offset+start+1+end
	}

	delete_from_data(data, to_delete)
	return lines
}

func delete_from_data(data *[]byte, pos [][]int) {
	alt := *data

	for i := len(pos)-1; i >= 0; i-- {
		tmp := make([]byte, 0)
		tmp = append(tmp, alt[:pos[i][0]]...)
		tmp = append(tmp, alt[pos[i][1]:]...)
		alt = tmp
	}

	*data = alt
}

func extract_pair(line *[]byte, equals []byte, term []byte) (string, string, bool) {
	eq := bytes.Index(*line, equals)
	if eq == -1 { return "", "", false }

	key := string((*line)[:eq])
	*line = (*line)[eq+len(equals):]

	space := bytes.Index(*line, term)
	var value string
	if space == -1 {
		value = string(*line)
		*line = []byte("")
	} else {
		value = string((*line)[:space])
		*line = (*line)[space+len(term):]
	}

	return key, value, true
}

func slash_numbers(s string) []uint64 {
	nums := make([]uint64, 0)

	for _, part := range strings.Split(s, "/") {
		base := 10
		if strings.HasPrefix(part, "0x") { base = 16 }

		// remove non-numbers
		part = strings.Map(
			func (r rune) rune {
				if unicode.IsNumber(r) { return r } else { return -1 }
			}, part)

		num, err := strconv.ParseUint(part, base, 64)
		if err != nil {
			log.Warningf("Cannot extract numbers from slash format \"%s\": %s", s, err)
		}
		nums = append(nums, num)
	}

	return nums
}

func find_lines_with_field(data *[]byte, field []byte, offsets *[]int) ([][]byte) {
	lines := make([][]byte, 0)
	start := 0
	cycle := 0

	for {
		// coding error protection
		cycle += 1
		if cycle > 100 {
			log.Errorf("Loop while looking for \"%s\" in %s", string(field), string(*data))
			return lines
		}

		//
		eq := bytes.Index((*data)[start:], field)
		if eq == -1 { break }

		// must start at the new line or after space
		if eq != 0 && (*data)[start+eq-1] != ' ' && (*data)[start+eq-1] != '\n' {
			start += eq + 1
			continue
		}

		// must end with some field delimiter
		end       := len((*data)[start+eq+1:])
		delimiter := []byte("=\n:")
		for _, delim := range delimiter {
			d := bytes.Index((*data)[start+eq+1:], []byte{delim})
			if d > 0 && d < end { end = d }
		}

		// if this is not what we are looking for, continue
		found := (*data)[start+eq:start+eq+end+1]
		if bytes.Compare(found, field) != 0 {
			start += eq+1
			continue
		}

		// find start and end of the line
		line_end   := bytes.IndexByte((*data)[start+eq:], '\n')
		if line_end   == -1 { line_end   = len((*data)[start+eq:]) }
		line_start := bytes.LastIndexByte((*data)[:start+eq], '\n')
		if line_start == -1 { line_start = 0 } else { line_start += 1 }

		if offsets != nil { *offsets = append(*offsets, start+eq-line_start) }

		line := (*data)[line_start:start+eq+line_end]
		line = bytes.TrimSpace(line)
		lines = append(lines, line)

		//
		start += eq+1
	}

	return lines
}
