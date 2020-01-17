// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package main

import (
	"fmt"
	"fortisession"
	"strings"
	"plugin"
	"os"
	"path"
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
	var pluginspec, data string
	var err error
	var pi pluginInfo

	// plugin pluginspec is the string before |, data the string after
	d := strings.Index(s, "|")
	if d == -1 {
		pluginspec = s
		data = ""
	} else {
		pluginspec = s[:d]
		data = s[d+1:]
	}

	// plugin file
	filename, err := search_plugin(pluginspec)
	if err != nil { return nil, fmt.Errorf("cannot local plugin file: %s", err) }

	p, err := plugin.Open(filename)
	if err != nil { return nil, fmt.Errorf("cannot load plugin file: %s", err) }

	pf, err := p.Lookup("InitPlugin")
	if err != nil { return nil, fmt.Errorf("cannot find \"InitPlugin\" function in plugin: %s", err) }

	pfinit, ok := pf.(func(string, *fortisession.SessionDataRequest)(map[string]func(*fortisession.Session)(bool),error))
	if !ok { return nil, fmt.Errorf("cannot verify type of \"InitPlugin\" function") }

	// init returns functions of the plugin
	refs, err := pfinit(data, data_request)
	if err != nil { return nil, fmt.Errorf("cannot initialize plugin: %s", err) }

	for k, v := range refs {
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

func search_plugin(s string) (string, error) {
	// if s starts with "/", we expect the full and exact path
	if s[0] == '/' { return s, nil }

	// otherwise we expect just a name of the plugin without .so at the end
	// and path is in "FOSET_PLUGINS" env, paths divided by ":"
	// of it is empty, default is $HOME/.foset/plugins
	env := os.Getenv("FOSET_PLUGINS")
	if len(env) == 0 { env = path.Join(os.Getenv("HOME"), ".foset", "plugins") }
	for _, dir := range strings.Split(env, ":") {
		fullname := path.Join(dir, s + ".so")
		log.Debugf("Checking for plugin in \"%s\"", fullname)
		_, staterr := os.Stat(fullname)
		if staterr != nil { continue }
		return fullname, nil
	}

	// not found, return error
	return "", fmt.Errorf("plugin not found")
}
