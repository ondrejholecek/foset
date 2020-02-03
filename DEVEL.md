# How to build the binary package

## Requirements

These are tested environments that I use. Others will most probably work as well as long as you have non-historical version of Go.

- Debian Buster or MacOS
- Go version at least 1.11.6

## Dependencies

```
$ go get github.com/akamensky/argparse
$ go get github.com/hprose/hprose-golang/io
$ go get github.com/juju/loggo
$ go get github.com/pkg/profile
$ go get golang.org/x/crypto/ssh
$ go get golang.org/x/crypto/ssh/agent
$ go get golang.org/x/crypto/ssh/terminal
```

### Necessary binary to build the go files from static data

```
$ go get -u github.com/shuLhan/go-bindata/...
```
