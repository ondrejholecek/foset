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

func Init(name, params string, custom_log loggo.Logger) (iprovider_common.IProvider, error) {
	log = custom_log
	log.Debugf("Initializing with \"%s\" params", params)

	ip := IProviderFile{
		name: name,
	}
	return ip, nil
}

func (ip IProviderFile) Name() (string) {
	return ip.name
}

func (ip IProviderFile) WaitReady() (error) {
	return nil
}

func (ip IProviderFile) CanProvideReader(name string) (bool, int) {
	if name == "" { return false, 0 }
	if name[0] == '/' { return true, 1000 }
	if name    == "-" { return true, 1000 }
	if strings.HasPrefix(name, "file://") { return true, 100000 }

	return false, 0
}

func (ip IProviderFile) CanProvideWriter(name string) (bool, int) {
	if name == "" { return false, 0 }
	if name[0] == '/' { return true, 1000 }
	if name    == "-" { return true, 1000 }
	if strings.HasPrefix(name, "file://") { return true, 100000 }

	return false, 0
}

func (ip IProviderFile) ProvideReader(name string) (io.Reader, *iprovider_common.ReaderParams, error) {
	var reader *os.File
	var params iprovider_common.ReaderParams
	var err    error

	if name == "-" {
		reader = os.Stdin
	} else {
		if strings.HasPrefix(name, "file://") {
			name = name[len("file://"):]
		}
		reader, err = os.Open(name)
		if err != nil { return nil, nil, err }
	}

	fi, _ := reader.Stat();
	params.IsTerminal = !(fi.Mode() & os.ModeCharDevice == 0)

	return reader, &params, nil
}

func (ip IProviderFile) ProvideWriter(name string) (io.Writer, *iprovider_common.WriterParams, error) {
	var writer *os.File
	var params iprovider_common.WriterParams
	var err    error

	if name == "-" {
		writer = os.Stdout
	} else {
		if strings.HasPrefix(name, "file://") {
			name = name[len("file://"):]
		}
		writer, err = os.Create(name)
		if err != nil { return nil, nil, err }
	}

	fi, _ := writer.Stat();
	params.IsTerminal = !(fi.Mode() & os.ModeCharDevice == 0)

	return writer, &params, nil
}
