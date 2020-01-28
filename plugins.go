// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package main

import (
	"fmt"
	"foset/fortisession"
	"strings"
	"plugin"
	"os"
	"path"
	"github.com/juju/loggo"
	"foset/plugins/common"
	// internal plugins:
	"foset/plugins/example"
	"foset/plugins/merge"
	"foset/plugins/stats"
	"foset/plugins/indexmap"
)

type pluginHook int
const (
	PLUGINS_BEFORE_FILTER       pluginHook = iota
	PLUGINS_AFTER_FILTER
	PLUGINS_FINISHED
)

// ************************
// *** external plugins ***
// ************************
func load_external_plugin(s string, data_request *fortisession.SessionDataRequest) (*plugin_common.FosetPlugin, error) {
	var err error

	pluginspec, data := split_plugin_name_data(s)

	// plugin file
	filename, err := search_plugin(pluginspec)
	if err != nil { return nil, fmt.Errorf("cannot local plugin file: %s", err) }

	p, err := plugin.Open(filename)
	if err != nil { return nil, fmt.Errorf("cannot load plugin file: %s", err) }

	pf, err := p.Lookup("InitPlugin")
	if err != nil { return nil, fmt.Errorf("cannot find \"InitPlugin\" function in plugin: %s", err) }

	pfinit, ok := pf.(func(string, *fortisession.SessionDataRequest, loggo.Logger)(*plugin_common.FosetPlugin, error))
	if !ok { return nil, fmt.Errorf("cannot verify type of \"InitPlugin\" function") }

	// init returns functions of the plugin
	var pluginInfo *plugin_common.FosetPlugin
	pluginInfo, err = pfinit(data, data_request, log.Child("eplugin"))
	if err != nil {
		return nil, fmt.Errorf("cannot initialize external plugin: %s", err)
	}

	//
	return pluginInfo, nil
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

// ************************
// *** internal plugins ***
// ************************

func load_internal_plugin(s string, data_request *fortisession.SessionDataRequest) (*plugin_common.FosetPlugin, error) {
	var err error
	pluginspec, data := split_plugin_name_data(s)

	// run init and get plugin info
	var pluginInfo *plugin_common.FosetPlugin

	if pluginspec == "merge" {
		pluginInfo, err = plugin_merge.InitPlugin(data, data_request, log.Child("iplugin"))
	} else if pluginspec == "stats" {
		pluginInfo, err = plugin_stats.InitPlugin(data, data_request, log.Child("iplugin"))
	} else if pluginspec == "indexmap" {
		pluginInfo, err = plugin_indexmap.InitPlugin(data, data_request, log.Child("iplugin"))
	} else if pluginspec == "example" {
		pluginInfo, err = plugin_example.InitPlugin(data, data_request, log.Child("iplugin"))
	} else {
		return nil, fmt.Errorf("unknown internal plugin: %s", pluginspec)
	}

	if err != nil {
		return nil, fmt.Errorf("cannot initialize internal plugin: %s", err)
	}

	return pluginInfo, nil
}

// ********************************
// *** generic plugin functions ***
// ********************************
func split_plugin_name_data(full string) (spec string, data string) {
	// plugin pluginspec is the string before |, data the string after
	d := strings.Index(full, "|")
	if d == -1 {
		spec = full
		data = ""
	} else {
		spec = full[:d]
		data = full[d+1:]
	}
	return
}

func run_plugins(plugins []*plugin_common.FosetPlugin, place pluginHook, session *fortisession.Session) bool {
	var ignore bool

	for _, plugin := range plugins {
		var r bool
		if        place == PLUGINS_BEFORE_FILTER && plugin.Hooks.BeforeFilter != nil {
			r = plugin.Hooks.BeforeFilter(session)
		} else if place == PLUGINS_AFTER_FILTER  && plugin.Hooks.AfterFilter  != nil {
			r = plugin.Hooks.AfterFilter(session)
		} else if place == PLUGINS_FINISHED      && plugin.Hooks.Finished     != nil {
			plugin.Hooks.Finished()
		}
		if r == true { ignore = true }
	}

	return ignore
}

func init_plugins() ([]*plugin_common.FosetPlugin) {
	return make([]*plugin_common.FosetPlugin, 0)
}
