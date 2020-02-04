// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package iprovider_ssh

import (
	"io"
	"os"
	"net"
	"fmt"
	"strings"
	"bytes"
	"strconv"
	"regexp"
	"time"
	"foset/iproviders/common"
	"foset/common"
	"github.com/juju/loggo"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/agent"
	"golang.org/x/crypto/ssh/terminal"
)

var log loggo.Logger

type connectionStatus int
const (
	CONNECTION_STATUS_IDLE             connectionStatus = iota
	CONNECTION_STATUS_INIT
	CONNECTION_STATUS_SIMPLE_COMMAND
	CONNECTION_STATUS_LONG_COMMAND
)

type FortiGateInfo struct {
	device     string
	model      string
	version    [3]uint64
	build      uint64
	mgmtVdom   string
	vdomMode   bool
}

type IProviderSsh struct {
	name    string
	sshc    *ssh.Client
	info    *FortiGateInfo
	status  connectionStatus
	last    time.Time
}

func Init(name, params string, custom_log loggo.Logger) (iprovider_common.IProvider, error) {
	log = custom_log
	log.Debugf("Initializing with \"%s\" params", params)

	// parse data parameters
	// use "host" parameter to turn this plugin on
	defaults := make(map[string]string)
	defaults["port"]      = "22"
	defaults["user"]      = "admin"
	defaults["password"]  = ""
	defaults["keepalive"] = "45"
	dk, du, _ := common.ExtractData(params, []string{"host","user","password","ask","port","agent","keepalive"}, defaults)

	// validate parameters
	unknowns := make([]string, 0)
	for k, _ := range du { unknowns = append(unknowns, k) }
	if len(unknowns) > 0 {
		return nil, fmt.Errorf("following parameters are not recognized: %s", strings.Join(unknowns, ", "))
	}

	var exists bool
	var err    error

	host, exists := dk["host"]
	if !exists {
		log.Debugf("Parameter \"host\" missing -> disabling plugin")
		return nil, nil
	}

	port, err := strconv.ParseUint(dk["port"], 10, 16)
	if err != nil { return nil, fmt.Errorf("parameter \"port\" invalid: %s", err) }

	keepalive, err := strconv.ParseUint(dk["keepalive"], 10, 16)
	if err != nil { return nil, fmt.Errorf("parameter \"keepalive\" invalid: %s", err) }

	_, use_agent   := dk["agent"]
	_, ask         := dk["ask"]

	// are we supposed to ask for password on command line?
	password  := dk["password"]
	if ask {
		fmt.Printf("Enter SSH password for %s@%s (port %d) : ", dk["user"], host, port)
		tmp, err := terminal.ReadPassword(0)
		fmt.Printf("\n")
		if err != nil { return nil, fmt.Errorf("cannot read ssh password: %s", err) }
		password = string(tmp)
	}

	// ssh connect
	var sshc *ssh.Client

	if use_agent {
		sshc, err = connectAgent(host, dk["user"], uint16(port), os.Getenv("SSH_AUTH_SOCK"))
	} else {
		sshc, err = connectPassword(host, dk["user"], uint16(port), password)
	}

	if err != nil { return nil, err }
	fgtInfo, err := getBasicInfo(sshc)
	if err != nil { return nil, err }

	ip := IProviderSsh{
		name   : name,
		sshc   : sshc,
		info   : fgtInfo,
		status : CONNECTION_STATUS_IDLE,
		last   : time.Now(),
	}

	if keepalive > 0 {
		log.Debugf("Starting keepalive when idle for %d seconds", keepalive)
		go ip.keepAlive(float64(keepalive))
	} else {
		log.Debugf("Not starting keepalive")
	}

	return ip, nil
}

func (ip IProviderSsh) Name() (string) {
	return ip.name
}

func (ip IProviderSsh) WaitReady() (error) {
	if ip.sshc == nil {
		return fmt.Errorf("ssh server is not connected")
	}

	for ip.status != CONNECTION_STATUS_IDLE {
		log.Debugf("Not ready because plugin is not idle")
		time.Sleep(time.Second)
	}
	log.Debugf("Plugin is ready")
	return nil
}

func (ip IProviderSsh) CanProvideReader(name string) (bool, int) {
	if strings.HasPrefix(name, "ssh://") { return true, 100000 }

	return false, 0
}

func (ip IProviderSsh) CanProvideWriter(name string) (bool, int) {
	return false, 0
}

func (ip IProviderSsh) ProvideReader(name string) (io.Reader, *iprovider_common.ReaderParams, error) {
	if !strings.HasPrefix(name, "ssh://") {
		return nil, nil, fmt.Errorf("cannot provide output for resource \"%s\"", name)
	}

	name = name[len("ssh://"):]

	// aliases
	if name == "vdoms"             { name = "<global>/simple/diagnose/sys/vd/list"
	} else if name == "interfaces" { name = "<mgmt>/simple/diagnose/netlink/interface/list"
	} else if name == "sessions"   { name = "<mgmt>/long/diagnose/sys/session/list"
	}

	// break it into parts
	parts := strings.Split(name, "/")
	if len(parts) < 3 {
		return nil, nil, fmt.Errorf("invalid resources \"%s\"", name)
	}

	vdom    := parts[0]
	cmdtype := parts[1]
	cmd     := strings.Join(parts[2:], " ")

	log.Debugf("Parsed resource: vdom=%s, cmdtype=%s, cmd=%s", vdom, cmdtype, cmd)

	var params iprovider_common.ReaderParams
	params.IsTerminal = false

	if cmdtype == "simple" {
		reader, err := ip.getSimpleCommand(cmd, vdom)
		return reader, &params, err

	} else if cmdtype == "long" {
		reader, err := ip.getLongCommand(cmd, vdom)
		return reader, &params, err

	} else {
		return nil, nil, fmt.Errorf("invalid command type \"%s\"", cmdtype)
	}

	// never gets here
}

func (ip IProviderSsh) ProvideWriter(name string) (io.Writer, *iprovider_common.WriterParams, error) {
	return nil, nil, fmt.Errorf("writer not implemented")
}

func (ip *IProviderSsh) keepAlive(idle float64) {
	for {
		if ip.sshc != nil && time.Now().Sub(ip.last).Seconds() > float64(idle) && ip.status == CONNECTION_STATUS_IDLE {
			log.Debugf("Ping due to inactivity")
			_, err := ip.getSimpleCommand("", "<global>")
			if err != nil {
				log.Debugf("Ping error: %s", err)
			} else {
				log.Debugf("Ping ok")
			}
		}
	}
}

func connectAgent(host, user string, port uint16, agentSocket string) (*ssh.Client, error) {
	log.Debugf("Connecting to %s:%d as \"%s\" with agent socket in \"%s\"", host, port, user, agentSocket)
	var err error

	conn_agent, err := net.Dial("unix", agentSocket)
	if err != nil { return nil, fmt.Errorf("cannot connect to agent: %s", err) }
	agentClient := agent.NewClient(conn_agent)

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeysCallback(agentClient.Signers),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshc, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil { return nil, fmt.Errorf("cannot connect to ssh server: %s", err) }

	log.Debugf("Connected to %s:%d as \"%s\"", host, port, user)
	return sshc, nil
}

func connectPassword(host, user string, port uint16, password string) (*ssh.Client, error) {
	log.Debugf("Connecting to %s:%d as \"%s\" with password", host, port, user)
	var err error

	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.Password(password),
		},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(),
	}

	sshc, err := ssh.Dial("tcp", fmt.Sprintf("%s:%d", host, port), config)
	if err != nil { return nil, fmt.Errorf("cannot connect to ssh server: %s", err) }

	log.Debugf("Connected to %s:%d as \"%s\"", host, port, user)
	return sshc, nil
}

func getBasicInfo(sshc *ssh.Client) (*FortiGateInfo, error) {
	session, err := sshc.NewSession()
	if err != nil { return nil, err }

	out, err := session.Output("get system status")

	// ... Version: FortiGate-1500D v6.2.1,build0932,
	// Virtual domain configuration: disable

	//r := regexp.MustCompile("Version:\\s+([^ ]+)\\s+v([0-9.]+),build([0-9]+),.*?Virtual\\s+domain\\s+configuration:\\s(.*?)\\s")
	rstr := "(?sm)"
	rstr += "(?:^|\\s)Version:\\s+([^-]+)-([^ ]+)\\s+v([0-9]+)\\.([0-9]+)\\.([0-9]+),build([0-9]+),.*?"
	rstr += "^Current\\s+virtual\\s+domain:\\s+(.*?)\\s.*?"
	rstr += "^Virtual\\s+domain\\s+configuration:\\s+(.*?)\\s.*?"
	r := regexp.MustCompile(rstr)
	matches := r.FindSubmatch(out)
	// FortiGate 1500D 6 2 1 0932 root disable
	if len(matches) != 9 {
		return nil, fmt.Errorf("cannot parse \"get system status\"")
	}

	var fgt FortiGateInfo
	fgt.device          = string(matches[1])
	fgt.model           = string(matches[2])
	fgt.version[0], _   = strconv.ParseUint(string(matches[3]), 10, 64)
	fgt.version[1], _   = strconv.ParseUint(string(matches[4]), 10, 64)
	fgt.version[2], _   = strconv.ParseUint(string(matches[5]), 10, 64)
	fgt.build, _        = strconv.ParseUint(string(matches[6]), 10, 64)
	fgt.mgmtVdom        = string(matches[7])
	if matches[8][0] == 'e' { fgt.vdomMode = true }

	return &fgt, nil
}

func (ip *IProviderSsh) constructCommand(command string, vdom string) (string) {
	cmd := ""

	if ip.info.vdomMode {
		if vdom == "" || vdom == "<global>" {
			cmd += "config global\n"

		} else if vdom == "<mgmt>" {
			cmd += fmt.Sprintf("config vdom\nedit %s\n", ip.info.mgmtVdom)

		} else {
			cmd += fmt.Sprintf("config vdom\nedit %s\n", vdom)
		}
	}

	cmd += command
	return cmd
}

// Runs a simple command and get Reader with the response
// vdom parameter: "" or "<global>"  - global
// vdom parameter: "<mgmt>"          - current management vdom
//                 anything else     - vdom name
func (ip *IProviderSsh) getSimpleCommand(command string, vdom string) (io.Reader, error) {
	ip.status = CONNECTION_STATUS_SIMPLE_COMMAND

	session, err := ip.sshc.NewSession()
	if err != nil { return nil, err }

	cmd := ip.constructCommand(command, vdom)
	log.Debugf("executing simple command \"%s\"", cmd)

	out, err := session.Output(cmd)
	if err != nil { return nil, err }
	session.Close()
	log.Tracef("command output: \"%s\"", out)

	ip.status = CONNECTION_STATUS_IDLE
	ip.last   = time.Now()

	reader := bytes.NewReader(out)
	return reader, nil
}

// runs a long command with io.Reader being filled in a gorutine and returns quickly
// vdom parameter same as for getSimpleCommand
func (ip *IProviderSsh) getLongCommand(command string, vdom string) (io.Reader, error) {
	ip.status = CONNECTION_STATUS_LONG_COMMAND

	session, err := ip.sshc.NewSession()
	if err != nil { return nil, err }

	cmd := ip.constructCommand(command, vdom)
	log.Debugf("executing long command \"%s\"", cmd)

	reader, err := session.StdoutPipe()
	if err != nil { return nil, err }

	err = session.Start(cmd)
	if err != nil { return nil, err }

	// wait for termination in gorutine
	go func() {
		log.Debugf("waiting for long command to finish")
		session.Wait()
		log.Debugf("long command finished, closing session")
		session.Close()

		ip.status = CONNECTION_STATUS_IDLE
		ip.last   = time.Now()
	}()

	return reader, nil
}
