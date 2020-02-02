package iprovider_file

import (
	"io"
	"os"
	"strings"
	"foset/iproviders/common"
	"github.com/juju/loggo"
)

var log loggo.Logger

type IProviderFile struct {
	name    string
}

func Init(name, params string, custom_log loggo.Logger) (iprovider_common.IProvider) {
	log = custom_log
	log.Debugf("Initializing with \"%s\" params", params)

	ip := IProviderFile{
		name: name,
	}
	return ip
}

func (ip IProviderFile) Name() (string) {
	return ip.name
}

func (ip IProviderFile) WaitReady() (error) {
	return nil
}

func (ip IProviderFile) CanProvide(name string) (bool, int) {
	if name == "" { return false, 0 }
	if name[0] == '/' { return true, 1000 }
	if name    == "-" { return true, 1000 }
	if strings.HasPrefix(name, "file://") { return true, 100000 }

	return false, 0
}

func (ip IProviderFile) ProvideResource(name string) (io.Reader, error) {
	var reader io.Reader
	var err    error

	if name == "-" {
		reader = os.Stdin
	} else {
		if strings.HasPrefix(name, "file://") {
			name = name[len("file://"):]
		}
		reader, err = os.Open(name)
		if err != nil { return nil, err }
	}

	return reader, nil
}
