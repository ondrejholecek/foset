package iprovider_common

import (
	"io"
)

type IProvider interface {
	Name() (string)
	WaitReady() (error)
	ProvideResource(name string) (io.Reader, error)
	CanProvide(name string) (bool, int)
}

