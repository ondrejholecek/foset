// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package main

import (
	"github.com/akamensky/argparse"
	"github.com/juju/loggo"
	"strings"
	"os"
	"fmt"
	"time"
	"runtime"
	"foset/plugins/common"
	"foset/fortisession"
	"foset/fortisession/fortiformatter"
	"foset/fortisession/forticonditioner"
	"foset/iproviders"
	"github.com/pkg/profile"
)

var mainVersion string
var fosetGitCommit          string

var log = loggo.GetLogger("foset")
var inputs *iproviders.IProviders

func main() {
	// read arguments
	parser := argparse.NewParser("foset", "Parses the FortiOS session list. Written by Ondrej Holecek <oholecek@fortinet.com>.")
	version    := parser.Flag(  "v", "version",  &argparse.Options{Default: false,            Help: "Print current version"})
	sessionfile:= parser.String("r", "file",     &argparse.Options{Default: "",               Help: "File containing the session list, use \"-\" for stdin"})
	output     := parser.String("o", "output",   &argparse.Options{Default: "${default_basic}", Help: "Format of the output"})
	filter     := parser.String("f", "filter",   &argparse.Options{Default: "",               Help: "Show only sessions matching filter"})
	debug      := parser.Flag(  "d", "debug",    &argparse.Options{Default: false,            Help: "Print also debugging outputs"})
	gzip_in    := parser.Flag(  "g", "gzip",     &argparse.Options{Default: false,            Help: "Enable if input is gzip compressed"})
	cache_save := parser.Flag(  "s", "save",     &argparse.Options{Default: false,            Help: "Only save parsed data to cache file [EXPERIMENTAL]"})
	cache_read := parser.Flag(  "c", "cache",    &argparse.Options{Default: false,            Help: "Load session data from cached file [EXPERIMENTAL]"})
	plugin_ext := parser.List(  "P", "external-plugin", &argparse.Options{                    Help: "Load external plugin library"})
	plugin_int := parser.List(  "p", "internal-plugin", &argparse.Options{                    Help: "Load internal plugin"})
	ipparams   := parser.List(  "i", "input-provider",  &argparse.Options{                    Help: "Parameters for input providers"})
	threads    := parser.Int(   "t", "threads",  &argparse.Options{Default: runtime.NumCPU(), Help: "Number of paralel threads to run, defaults to number of available cores"})
	loop       := parser.Int(   "l", "loop",     &argparse.Options{Default: 1,                Help: "Number of cycles to run, zero for infinite loop"})
	loop_time  := parser.Int(   "",  "loop-time",&argparse.Options{Default: 1,                Help: "How often to repeat the cycle (seconds, including the execution time)"})
	nobuffer   := parser.Flag(  "n", "no-buffer",&argparse.Options{Default: false,            Help: "Disable output buffering"})
	trace      := parser.Flag(  "", "trace",     &argparse.Options{Default: false,            Help: "Debugging: enable trace outputs"})
	parse_all  := parser.Flag(  "", "parse-all", &argparse.Options{Default: false,            Help: "Debugging: parse all fields regardless on filter and output"})
	profiler   := parser.String(  "", "profiler",&argparse.Options{Default: "",               Help: "Debugging: enable profiler (mem or cpu)"})
	if err := parser.Parse(os.Args); err != nil {
		fmt.Println(err)
		fmt.Println("Use -h for help")
		os.Exit(1)
	}

	if (*version) {
		fmt.Printf("----------------------\n")
		fmt.Printf("FOrtigate SEssion Tool\n")
		fmt.Printf("----------------------\n")
		fmt.Printf("Written by Ondrej Holecek <ondrej@holecek.eu>\n")
		fmt.Printf("Version: %s\n\n", mainVersion)
		fmt.Printf("This software is governed by the CC BY-ND 4.0 license.\n")
		fmt.Printf("Make sure you understand the license before modifying the code!\n")
		fmt.Printf("(https://creativecommons.org/licenses/by-nd/4.0/)\n\n")
		fmt.Printf("Git commit : %s\n", fosetGitCommit)
		os.Exit(0)
	}

	if *sessionfile == "" {
		fmt.Println("File parameter required\nUse -h for help")
		os.Exit(1)
	}

	if *profiler == "cpu" {
		defer profile.Start().Stop()
	} else if *profiler == "mem" {
		defer profile.Start(profile.MemProfile).Stop()
	} else if len(*profiler) > 0 {
		fmt.Println("Unknown profile type")
		os.Exit(1)
	}

	//
	if (*trace) {
		log.SetLogLevel(loggo.TRACE)
	} else if (*debug) {
		log.SetLogLevel(loggo.DEBUG)
	} else {
		log.SetLogLevel(loggo.INFO)
	}
	forticonditioner.InitLog(log.Child("forticonditioner"))
	fortiformatter.InitLog(log.Child("formatter"))
	fortisession.InitLog(log.Child("session"))

	// input providers
	var err error
	inputs, err = iproviders.Init(*ipparams, log.Child("iproviders"))
	if err != nil {
		log.Criticalf("cannot initialize providers: %s", err)
		os.Exit(100)
	}

	//
	data_request := fortisession.SessionDataRequest {}

	// plugins - generic
	plugins := init_plugins()

	// plugins - external
	for _, p := range *plugin_ext {
		log.Debugf("Loading external plugin \"%s\"", p)
		pinfo := plugin_common.FosetPlugin{
			Filename : *sessionfile,
			Filter   : *filter,
			Version  : mainVersion,
			Commit   : fosetGitCommit,
			Inputs   : inputs,
		}
		err := load_external_plugin(p, &data_request, &pinfo)
		if err != nil {
			log.Criticalf("Cannot load external plugin: %s", err)
			os.Exit(100)
		}
		plugins = append(plugins, &pinfo)
		log.Debugf("Done")
	}

	// plugins - internal
	for _, p := range *plugin_int {
		log.Debugf("Loading internal plugin \"%s\"", p)
		pinfo := plugin_common.FosetPlugin{
			Filename : *sessionfile,
			Filter   : *filter,
			Version  : mainVersion,
			Commit   : fosetGitCommit,
			Inputs   : inputs,
		}
		err := load_internal_plugin(p, &data_request, &pinfo)
		if err != nil {
			log.Criticalf("Cannot load internal plugin: %s", err)
			os.Exit(100)
		}
		plugins = append(plugins, &pinfo)
		log.Debugf("Done")
	}

	//
	formatter, err := fortiformatter.Init(*output, &data_request)
	if err != nil {
		log.Criticalf("Cannot parse output format: %s\n", err)
		os.Exit(100)
	}
	log.Debugf("Parser request struct after formatter init: %#v", data_request)

	var conditioner *forticonditioner.Condition = nil
	if len(*filter) > 0 {
		conditioner = forticonditioner.Init(*filter, &data_request)
	}
	log.Debugf("Parser request struct after conditioner init: %#v", data_request)

	if (*parse_all) {
		data_request.SetAll()
	}

	if len(*filter) > 0 {
		log.Debugf("Original filter: \"%s\"", *filter)
		log.Debugf("Parsed filter:")
		for _, l := range strings.Split(conditioner.DumpPretty(), "\n") {
			if len(l) == 0 { continue }
			log.Debugf("%s", l)
		}
	} else {
		log.Debugf("No filter specified")
	}

	//
	ep := ExecuteParams {
		threads        : *threads,
		sessionfile    : *sessionfile,
		nobuffer       : *nobuffer,
		gzip_in        : *gzip_in,
		cache_save     : *cache_save,
		cache_read     : *cache_read,
		data_request   : &data_request,
		conditioner    : conditioner,
		formatter      : formatter,
		plugins        : plugins,
	}


	for i := 0; i < *loop || *loop == 0; i++ {
		log.Debugf("Starting next cycle")
		start := time.Now()

		err := inputs.WaitReady()
		if err != nil {
			log.Criticalf("Input providers are not ready: %s", err)
			break
		}

		run_plugins(plugins, PLUGINS_START, nil)
		execute(ep)
		runtime.GC()

		took  := time.Now().Sub(start)
		log.Debugf("Last cycle took %.1f seconds", took.Seconds())

		// prevent sleep after last cycle
		if (i+1) >= *loop { continue }

		sleep := time.Duration(*loop_time) * time.Duration(time.Second) - took
		if sleep.Seconds() > 0 {
			log.Debugf("Will sleep for %.1f seconds", took.Seconds(), sleep.Seconds())
			time.Sleep(sleep)
		} else {
			log.Debugf("Cycle repeating immediately")
		}
	}
}

