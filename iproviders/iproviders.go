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

func (ips *IProviders) Provide(name string) (io.Reader, error) {
	log.Debugf("looking for input provider for \"%s\"", name)

	// first find all possible providers and use the one
	// with highest priority
	var best_provider iprovider_common.IProvider
	var best_priority int

	for _, ip := range ips.iproviders {
		can, prio := ip.CanProvide(name)
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

	// get stream from selected provider
	resource, err := best_provider.ProvideResource(name)
	if err != nil {
		return nil, fmt.Errorf("provider \"%s\" error: %s", best_provider.Name(), err)
	}

	return resource, nil
}

func (ips *IProviders) WaitReady() (error) {
	for _, ip := range ips.iproviders {
		err := ip.WaitReady()
		if err != nil { return err }
	}

	return nil
}
