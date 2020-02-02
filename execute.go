package main

import (
	"fmt"
	"bufio"
	"os"
	"foset/plugins/common"
	"foset/fortisession"
	"foset/fortisession/fortiformatter"
	"foset/fortisession/forticonditioner"
)

type ExecuteParams struct {
	threads       int
	sessionfile   string
	nobuffer      bool
	gzip_in       bool
	cache_save    bool
	cache_read    bool
	data_request  *fortisession.SessionDataRequest
	conditioner   *forticonditioner.Condition
	formatter     *fortiformatter.Formatter
	plugins       []*plugin_common.FosetPlugin
}

func execute(ep ExecuteParams) {
	parsed_sessions        := make(chan *fortisession.Session, 250*(ep.threads))
	all_sessions_collected := make(chan bool)

	var session_cache *CacheFile
	var inerr error

	if ep.cache_save {
		session_cache, inerr = CacheInit(ep.sessionfile+ ".cache", "w", ep.threads)
		ep.data_request.Plain = false
		go save_sessions(parsed_sessions, session_cache, ep.conditioner, ep.plugins, all_sessions_collected)
		file_processing := Init_file_processing(parsed_sessions, ep.data_request, ep.threads, ep.conditioner, ep.plugins)
		inerr = file_processing.Read_all_from_file(ep.sessionfile, Compression { Gzip : ep.gzip_in })

	} else if ep.cache_read {
		session_cache, inerr = CacheInit(ep.sessionfile+ ".cache", "r", ep.threads)
		go collect_sessions(parsed_sessions, ep.formatter, ep.conditioner, ep.plugins, all_sessions_collected, !(ep.nobuffer))
		inerr = session_cache.ReadAll(parsed_sessions)

	} else {
		go collect_sessions(parsed_sessions, ep.formatter, ep.conditioner, ep.plugins, all_sessions_collected, !(ep.nobuffer))
		file_processing := Init_file_processing(parsed_sessions, ep.data_request, ep.threads, ep.conditioner, ep.plugins)
		inerr = file_processing.Read_all_from_file(ep.sessionfile, Compression { Gzip : ep.gzip_in })
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

