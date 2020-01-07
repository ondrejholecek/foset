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
	output     := parser.String("o", "output",   &argparse.Options{Default: "${default_basic} ${default_hw}, ${default_rate}, ${default_counts}", Help: "Format of the output"})
	filter     := parser.String("f", "filter",   &argparse.Options{Default: "",               Help: "Show only sessions matching filter"})
	debug      := parser.Flag(  "d", "debug",    &argparse.Options{Default: false,            Help: "Print also debugging outputs"})
	gzip_in    := parser.Flag(  "g", "gzip",     &argparse.Options{Default: false,            Help: "Enable if input is gzip compressed"})
	cache_save := parser.Flag(  "s", "save",     &argparse.Options{Default: false,            Help: "Only save parsed data to cache file [EXPERIMENTAL]"})
	cache_read := parser.Flag(  "c", "cache",    &argparse.Options{Default: false,            Help: "Load session data from cached file [EXPERIMENTAL]"})
	threads    := parser.Int(   "t", "threads",  &argparse.Options{Default: runtime.NumCPU(), Help: "Number of paralel threads to run, defaults to number of available cores"})
	nobuffer   := parser.Flag(  "", "no-buffer", &argparse.Options{Default: false,            Help: "Disable output buffering"})
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
		fmt.Printf("Written by Ondrej Holecek <ondrej@holecek.eu>\n\n")
		fmt.Printf("This software is governed by the CC BY-ND 4.0 license.\n")
		fmt.Printf("Make sure you understand the license before modifying the code!\n")
		fmt.Printf("(https://creativecommons.org/licenses/by-nd/4.0/)\n\n")
		fmt.Printf("Main program version: %s %s\n", mainVersion, fosetGitCommit)
		fmt.Printf("Fortisession library: %s\n", fortisessionGitCommit)
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
		go save_sessions(parsed_sessions, session_cache, conditioner, all_sessions_collected)
		file_processing := Init_file_processing(parsed_sessions, &data_request, *threads)
		inerr = file_processing.Read_all_from_file(*filename, Compression { Gzip : *gzip_in })

	} else if *cache_read {
		session_cache, inerr = CacheInit(*filename + ".cache", "r", *threads)
		go collect_sessions(parsed_sessions, formatter, conditioner, all_sessions_collected, !(*nobuffer))
		inerr = session_cache.ReadAll(parsed_sessions)

	} else {
		go collect_sessions(parsed_sessions, formatter, conditioner, all_sessions_collected, !(*nobuffer))
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


func save_sessions(results chan *fortisession.Session, cache *CacheFile, conditioner *forticonditioner.Condition, done chan bool) {
	for session := range results {
		if conditioner != nil && !conditioner.Matches(session) { continue }
		cache.Write(session)
	}
	done <- true
}

func collect_sessions(results chan *fortisession.Session, formatter *fortiformatter.Formatter, conditioner *forticonditioner.Condition, done chan bool, buffer bool) {
	// prepare the buffer (even if it is not going to be used)
	w := bufio.NewWriterSize(os.Stdout, 16384)
	defer w.Flush()

	//
	for session := range results {
		log.Tracef("Collecting session: %#x\n%#f", session.Serial, session)
		if conditioner != nil && !conditioner.Matches(session) { continue }

		if buffer {
			w.WriteString(formatter.Format(session) + "\n")
		} else {
			fmt.Println(formatter.Format(session))
		}
	}
	done <- true
}

