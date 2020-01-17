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
	"fortisession"
	"fortisession/fortiformatter"
	"fortisession/forticonditioner"
	"github.com/pkg/profile"
)

var mainVersion string
var fosetGitCommit          string
var fortisessionGitCommit   string

var log = loggo.GetLogger("foset")

func main() {
	// read arguments
	parser := argparse.NewParser("foset", "Parses the FortiOS session list. Written by Ondrej Holecek <oholecek@fortinet.com>.")
	version    := parser.Flag(  "v", "version",  &argparse.Options{Default: false,            Help: "Print current version"})
	filename   := parser.String("r", "file",     &argparse.Options{Default: "",               Help: "File containing the session list"})
	output     := parser.String("o", "output",   &argparse.Options{Default: "${default_basic}", Help: "Format of the output"})
	filter     := parser.String("f", "filter",   &argparse.Options{Default: "",               Help: "Show only sessions matching filter"})
	external   := parser.List  ("e", "external", &argparse.Options{                           Help: "Load additional data from external file (see documentation)"})
	debug      := parser.Flag(  "d", "debug",    &argparse.Options{Default: false,            Help: "Print also debugging outputs"})
	gzip_in    := parser.Flag(  "g", "gzip",     &argparse.Options{Default: false,            Help: "Enable if input is gzip compressed"})
	cache_save := parser.Flag(  "s", "save",     &argparse.Options{Default: false,            Help: "Only save parsed data to cache file [EXPERIMENTAL]"})
	cache_read := parser.Flag(  "c", "cache",    &argparse.Options{Default: false,            Help: "Load session data from cached file [EXPERIMENTAL]"})
	plugin_ext := parser.List(  "P", "external-plugin", &argparse.Options{                    Help: "Load external plugin library"})
	plugin_int := parser.List(  "p", "internal-plugin", &argparse.Options{                    Help: "Load internal plugin"})
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
		fmt.Printf("Main program         : %s\n", fosetGitCommit)
		fmt.Printf("Fortisession library : %s\n", fortisessionGitCommit)
		os.Exit(0)
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

	//
	data_request := fortisession.SessionDataRequest {}

	// plugins - generic
	plugins := make([]*pluginInfo, 0)

	// plugins - external
	for _, p := range *plugin_ext {
		log.Debugf("Loading external plugin \"%s\"", p)
		pinfo, err := load_external_plugin(p, &data_request)
		if err != nil {
			log.Criticalf("Cannot load external plugin: %s", err)
			os.Exit(100)
		}
		plugins = append(plugins, pinfo)
		log.Debugf("Done")
	}

	// plugins - internal
	init_internal_plugins()
	for _, p := range *plugin_int {
		log.Debugf("Loading internal plugin \"%s\"", p)
		pinfo, err := load_internal_plugin(p, &data_request)
		if err != nil {
			log.Criticalf("Cannot load internal plugin: %s", err)
			os.Exit(100)
		}
		plugins = append(plugins, pinfo)
		log.Debugf("Done")
	}

	// external (additional) data
	var custom_data []*forticonditioner.CustomData
	for _, e := range *external {
		log.Debugf("Loading external data \"%s\"", e)
		custom_data = append(custom_data, load_external(e))
		data_request.Custom = true
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
		conditioner = forticonditioner.Init(*filter, &data_request, custom_data)
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
		file_processing := Init_file_processing(parsed_sessions, &data_request, *threads)
		inerr = file_processing.Read_all_from_file(*filename, Compression { Gzip : *gzip_in })

	} else if *cache_read {
		session_cache, inerr = CacheInit(*filename + ".cache", "r", *threads)
		go collect_sessions(parsed_sessions, formatter, conditioner, plugins, all_sessions_collected, !(*nobuffer))
		inerr = session_cache.ReadAll(parsed_sessions)

	} else {
		go collect_sessions(parsed_sessions, formatter, conditioner, plugins, all_sessions_collected, !(*nobuffer))
		file_processing := Init_file_processing(parsed_sessions, &data_request, *threads)
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


func save_sessions(results chan *fortisession.Session, cache *CacheFile, conditioner *forticonditioner.Condition, plugins []*pluginInfo, done chan bool) {
	for session := range results {
		if run_plugins(plugins, PLUGINS_BEFORE_FILTER, session) { continue }
		if conditioner != nil && !conditioner.Matches(session) { continue }
		if run_plugins(plugins, PLUGINS_AFTER_FILTER, session) { continue }
		cache.Write(session)
	}
	run_plugins(plugins, PLUGINS_END, nil)
	done <- true
}

func collect_sessions(results chan *fortisession.Session, formatter *fortiformatter.Formatter, conditioner *forticonditioner.Condition, plugins []*pluginInfo, done chan bool, buffer bool) {
	// prepare the buffer (even if it is not going to be used)
	w := bufio.NewWriterSize(os.Stdout, 1024)

	//
	for session := range results {
		log.Tracef("Collecting session: %#x\n%#f", session.Serial, session)

		if run_plugins(plugins, PLUGINS_BEFORE_FILTER, session) { continue }

		if conditioner != nil && !conditioner.Matches(session) { continue }  // this seems to hit some bug in Go where memory start being shifted like hell
		                                                                     // and the program runs incredibly slow (because of garbage collectors (??) )
		                                                                     // ... it happens only when customdata (-e) are used
		                                                                     // ... it is not just this call, but also some nested calls
		                                                                     // TODO: fix
		if run_plugins(plugins, PLUGINS_AFTER_FILTER, session) { continue }

		if buffer {
			w.WriteString(formatter.Format(session) + "\n")
		} else {
			fmt.Println(formatter.Format(session))
		}
	}

	w.Flush()
	run_plugins(plugins, PLUGINS_END, nil)
	done <- true
}

func load_external(external string) (*forticonditioner.CustomData) {
	var custom forticonditioner.CustomData
	custom.Data = make(map[string][]string)

	// format: filename
	// or    : filename|delimiter|(keyfield)|other1|other2
	// or    : filename|delimiter|other1|(keyfield)|other2
	// etc...
	parts := strings.Split(external, "|")
	var filename   string     = parts[0]
	var delimiter  string
	var key        int

	if len(parts) >= 2 {
		delimiter         = parts[1]
		custom.FieldNames = parts[2:]
	}

	// for scanning in later code this cannot be empty
	if len(delimiter) == 0 { delimiter = " " }

	// find key name and remove parethesis
	for i, fieldName := range custom.FieldNames {
		if !strings.HasPrefix(fieldName, "(") || !strings.HasSuffix(fieldName, ")") { continue }
		custom.KeyName       = fieldName[1:len(fieldName)-1]
		custom.FieldNames[i] = custom.KeyName
		key                  = i
	}

	// we need a key name..
	if len(custom.KeyName) == 0 {
		log.Criticalf("External file specification does not contain a key field")
		os.Exit(1)
	}

	log.Tracef("External file: [%s], delimiter: [%s], key: [%s], fieds: [%f]\n", filename, delimiter, custom.KeyName, custom.FieldNames)

	// open the file
	f, err := os.Open(filename)
	if err != nil {
		log.Criticalf("Unable to open external file: %s", err)
		os.Exit(1)
	}

	// fill the data
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line  := scanner.Text()
		parts := strings.Split(line, delimiter)

		if len(custom.FieldNames) > len(parts) {
			log.Criticalf("External file does not have enough parts: \"%s\"", line)
			os.Exit(1)
		}

		custom.Data[parts[key]] = parts
	}

	return &custom
}
