package iproviders

import (
	"io"
	"fmt"
	"strings"
	"foset/iproviders/common"
	"foset/iproviders/file"
	"foset/iproviders/ssh"
	"github.com/juju/loggo"
)

var log loggo.Logger

type IProviders struct {
	iproviders []iprovider_common.IProvider
}

func Init(params []string, custom_log loggo.Logger) (*IProviders, error) {
	log = custom_log

	// create map of parameters
	pmap := make(map[string]string)
	for _, p := range params {
		n := strings.Index(p, "|")
		if n == -1 {
			log.Warningf("Ignoring input parameter \"%s\" because it does not contain \"|\" character", p)
			continue
		}

		pmap[p[:n]] = p[n+1:]
	}

	// initialize known providers
	// provider init function cab return nil and error -> we will display it and terminate
	// it can also return nil and nil -> plugin wants to be disabled
	ips   := IProviders {
		iproviders : make([]iprovider_common.IProvider, 0),
	}

	var p   iprovider_common.IProvider
	var err error

	p, err = iprovider_file.Init("file", pmap["file"], log.Child("file"))
	if err != nil { return nil, fmt.Errorf("cannot initialize provider \"file\": %s", err) }
	if p != nil   { ips.iproviders = append(ips.iproviders, p) }

	p, err = iprovider_ssh.Init("ssh", pmap["ssh"], log.Child("ssh"))
	if err != nil { return nil, fmt.Errorf("cannot initialize provider \"ssh\": %s", err) }
	if p != nil   { ips.iproviders = append(ips.iproviders, p) }

	return &ips, nil
}

func (ips *IProviders) ProvideReader(name string) (io.Reader, *iprovider_common.ReaderParams, error) {
	log.Debugf("looking for input provider reader for \"%s\"", name)

	// find provider
	provider, err := ips.findProvider(name, "r")
	if err != nil {
		return nil, nil, fmt.Errorf("find provider error: %s", err)
	}

	// get stream
	resource, params, err := provider.ProvideReader(name)
	if err != nil {
		return nil, nil, fmt.Errorf("provider reader \"%s\" error: %s", provider.Name(), err)
	}

	return resource, params, nil
}

func (ips *IProviders) ProvideWriter(name string) (io.Writer, *iprovider_common.WriterParams, error) {
	log.Debugf("looking for input provider writer for \"%s\"", name)

	// find provider
	provider, err := ips.findProvider(name, "w")
	if err != nil {
		return nil, nil, fmt.Errorf("find provider error: %s", err)
	}

	// get stream
	resource, params, err := provider.ProvideWriter(name)
	if err != nil {
		return nil, nil, fmt.Errorf("provider writer \"%s\" error: %s", provider.Name(), err)
	}

	return resource, params, nil
}

func (ips *IProviders) findProvider(name string, r_or_w string) (iprovider_common.IProvider, error) {
	// first find all possible providers and use the one
	// with highest priority
	var best_provider iprovider_common.IProvider
	var best_priority int

	for _, ip := range ips.iproviders {
		var can  bool
		var prio int

		if r_or_w == "r" {
			can, prio = ip.CanProvideReader(name)
		} else if r_or_w == "w" {
			can, prio = ip.CanProvideWriter(name)
		} else {
			return nil, fmt.Errorf("r_or_w parameter must be either \"r\" or \"w\"")
		}

		if !can      { continue }
		if prio <= 0 { continue }

		log.Debugf("provider \"%s\" can provide it with priority %d", ip.Name(), prio)
		if prio > best_priority {
			best_priority = prio
			best_provider = ip
		}
	}

	if best_provider == nil {
		return nil, fmt.Errorf("no provider found for \"%s\"", name)
	} else {
		log.Debugf("selected provider \"%s\"", best_provider.Name())
	}

	return best_provider, nil
}


func (ips *IProviders) WaitReady() (error) {
	for _, ip := range ips.iproviders {
		err := ip.WaitReady()
		if err != nil { return err }
	}

	return nil
}
