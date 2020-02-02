// Copyright 2020 Ondrej Holecek <ondrej@holecek.eu>. All rights reserved. Use of this source code
// is governed by the CC BY-ND 4.0 license that can be found in the LICENSE.txt file.

package main

import (
	"github.com/akamensky/argparse"
	"github.com/juju/loggo"
	"strings"
	"bufio"
	"os"
	"fmt"
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
	filename   := parser.String("r", "file",     &argparse.Options{Default: "",               Help: "File containing the session list, use \"-\" for stdin"})
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

	if *filename == "" {
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
	inputs = iproviders.Init(*ipparams, log.Child("iproviders"))

	//
	data_request := fortisession.SessionDataRequest {}

	// plugins - generic
	plugins := init_plugins()

	// plugins - external
	for _, p := range *plugin_ext {
		log.Debugf("Loading external plugin \"%s\"", p)
		pinfo := plugin_common.FosetPlugin{
			Filename : *filename,
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
			Filename : *filename,
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

	parsed_sessions        := make(chan *fortisession.Session, 250*(*threads))
	all_sessions_collected := make(chan bool)

	var session_cache *CacheFile
	var inerr error

	if *cache_save {
		session_cache, inerr = CacheInit(*filename + ".cache", "w", *threads)
		data_request.Plain = false
		go save_sessions(parsed_sessions, session_cache, conditioner, plugins, all_sessions_collected)
		file_processing := Init_file_processing(parsed_sessions, &data_request, *threads, conditioner, plugins)
		inerr = file_processing.Read_all_from_file(*filename, Compression { Gzip : *gzip_in })

	} else if *cache_read {
		session_cache, inerr = CacheInit(*filename + ".cache", "r", *threads)
		go collect_sessions(parsed_sessions, formatter, conditioner, plugins, all_sessions_collected, !(*nobuffer))
		inerr = session_cache.ReadAll(parsed_sessions)

	} else {
		go collect_sessions(parsed_sessions, formatter, conditioner, plugins, all_sessions_collected, !(*nobuffer))
		file_processing := Init_file_processing(parsed_sessions, &data_request, *threads, conditioner, plugins)
		inerr = file_processing.Read_all_from_file(*filename, Compression { Gzip : *gzip_in })
	}

	if inerr != nil {
		log.Criticalf("Input data read error: %s", inerr)
		os.Exit(100)
	}

	close(parsed_sessions)
	<-all_sessions_collected // wait for all sessions to be collected in gorutine before exiting main program
	if session_cache != nil { session_cache.Finalize() }
}


func save_sessions(results chan *fortisession.Session, cache *CacheFile, conditioner *forticonditioner.Condition, plugins []*plugin_common.FosetPlugin, done chan bool) {
	for session := range results {
		if run_plugins(plugins, PLUGINS_BEFORE_FILTER, session) { continue }
		if conditioner != nil && !conditioner.Matches(session) { continue }
		if run_plugins(plugins, PLUGINS_AFTER_FILTER, session) { continue }
		cache.Write(session)
	}
	run_plugins(plugins, PLUGINS_FINISHED, nil)
	done <- true
}

func collect_sessions(results chan *fortisession.Session, formatter *fortiformatter.Formatter, conditioner *forticonditioner.Condition, plugins []*plugin_common.FosetPlugin, done chan bool, buffer bool) {
	// prepare the buffer (even if it is not going to be used)
	w := bufio.NewWriterSize(os.Stdout, 1024)
	// is output terminal?
	fi, _ := os.Stdout.Stat();
	terminal := !(fi.Mode() & os.ModeCharDevice == 0)

	//
	for session := range results {
		log.Tracef("Collecting session: %#x\n%#f", session.Serial, session)

		if buffer && !terminal {
			w.WriteString(formatter.Format(session) + "\n")
		} else {
			fmt.Println(formatter.Format(session))
		}
	}

	w.Flush()
	run_plugins(plugins, PLUGINS_FINISHED, nil)
	done <- true
}
