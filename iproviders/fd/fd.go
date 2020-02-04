package iprovider_fd

import (
	"fmt"
	"strconv"
	"io"
	"os"
	"strings"
	"foset/iproviders/common"
	"github.com/juju/loggo"
)

var log loggo.Logger

type IProviderFd struct {
	name    string
}

func Init(name, params string, custom_log loggo.Logger) (*IProviderFd, error) {
	log = custom_log
	log.Debugf("Initializing with \"%s\" params", params)

	ip := &IProviderFd{
		name: name,
	}
	return ip, nil
}

func (ip IProviderFd) Name() (string) {
	return ip.name
}

func (ip IProviderFd) WaitReady() (error) {
	return nil
}

func (ip IProviderFd) CanProvideReader(name string) (bool, int) {
	if strings.HasPrefix(name, "fd://") { return true, 100000 }

	return false, 0
}

func (ip IProviderFd) CanProvideWriter(name string) (bool, int) {
	if strings.HasPrefix(name, "fd://") { return true, 100000 }

	return false, 0
}

func (ip *IProviderFd) ProvideReader(name string) (io.Reader, *iprovider_common.ReaderParams, error) {
	if !strings.HasPrefix(name, "fd://") {
		return nil, nil, fmt.Errorf("invalid resource name")
	}

	fd, err := strconv.ParseInt(name[len("fd://"):], 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse fd: \"%s\"", name)
	}

	f := os.NewFile(uintptr(fd), name)
	if f == nil { return nil, nil, fmt.Errorf("cannot attach fd %d", fd) }

	var params iprovider_common.ReaderParams
	fi, _ := f.Stat();
	if fi == nil {
		log.Errorf("Cannot stat reader fd %d", fd)
	} else {
		params.IsTerminal = !(fi.Mode() & os.ModeCharDevice == 0)
	}

	return f, &params, nil
}

func (ip *IProviderFd) ProvideWriter(name string) (io.Writer, *iprovider_common.WriterParams, error) {
	if !strings.HasPrefix(name, "fd://") {
		return nil, nil, fmt.Errorf("invalid resource name")
	}

	fd, err := strconv.ParseInt(name[len("fd://"):], 10, 64)
	if err != nil {
		return nil, nil, fmt.Errorf("unable to parse fd: \"%s\"", name)
	}

	f := os.NewFile(uintptr(fd), name)
	if f == nil { return nil, nil, fmt.Errorf("cannot attach fd %d", fd) }

	var params iprovider_common.WriterParams
	fi, _ := f.Stat();
	if fi == nil {
		log.Errorf("Cannot stat writer fd %d", fd)
	} else {
		params.IsTerminal = !(fi.Mode() & os.ModeCharDevice == 0)
	}

	return f, &params, nil
}
