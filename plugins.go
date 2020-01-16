// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package main

import (
	"fmt"
	"fortisession"
	"strings"
	"plugin"
)

type pluginInfo struct {
	beforeFilter   func(*fortisession.Session)(bool)
	afterFilter    func(*fortisession.Session)(bool)
	end            func(*fortisession.Session)(bool)
}

const (
	PLUGINS_BEFORE_FILTER         = iota
	PLUGINS_AFTER_FILTER
	PLUGINS_END
)

func load_external_plugin(s string, data_request *fortisession.SessionDataRequest) (*pluginInfo, error) {
	var path, data string
	var err error
	var pi pluginInfo

	// plugin path is the string before |, data the string after
	d := strings.Index(s, "|")
	if d == -1 {
		path = s
		data = ""
	} else {
		path = s[:d]
		data = s[d+1:]
	}

	// plugin file
	p, err := plugin.Open(path)
	if err != nil {
		return nil, fmt.Errorf("Cannot load plugin file: %s", err)
	}

	pf, err := p.Lookup("InitPlugin")
	if err != nil {
		return nil, fmt.Errorf("Cannot find \"InitPlugin\" function in plugin: %s", err)
	}

	pfinit, ok := pf.(func(string, *fortisession.SessionDataRequest)(map[string]func(*fortisession.Session)(bool)))
	if !ok {
		return nil, fmt.Errorf("Cannot verify type of \"InitPlugin\" function")
	}

	// init returns functions of the plugin
	for k, v := range pfinit(data, data_request) {
		if k == "beforeFilter" { pi.beforeFilter = v
		} else if k == "afterFilter"  { pi.afterFilter = v
		} else if k == "end"          { pi.end = v
		} else {
			log.Warningf("External plugin \"%s\" uses unknown trap \"%s\"", s, k)
		}
	}

	return &pi, nil
}

func run_plugins(plugins []*pluginInfo, place int, session *fortisession.Session) bool {
	var ignore bool

	for _, plugin := range plugins {
		var r bool
		if        place == PLUGINS_BEFORE_FILTER && plugin.beforeFilter != nil { r = plugin.beforeFilter(session)
		} else if place == PLUGINS_AFTER_FILTER  && plugin.afterFilter  != nil { r = plugin.afterFilter(session)
		} else if place == PLUGINS_END           && plugin.end          != nil { r = plugin.end(session)
		}
		if r == true { ignore = true }
	}

	return ignore
}
