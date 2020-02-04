MAIN_VERSION=$(shell cat VERSION.txt)
FOSET_GIT_COMMIT := $(shell git rev-list -1 HEAD)
LDFLAGS=-ldflags="-s -w -X main.fosetGitCommit=$(FOSET_GIT_COMMIT) -X main.mainVersion=$(MAIN_VERSION)"
RELEASE_DIR=release/$(MAIN_VERSION)
LATEST_DIR=release/latest

all: local
release: macos linux windows

macos:
	mkdir -p $(RELEASE_DIR)/macos
	GOOS=darwin  GOARCH=amd64 go build -o $(RELEASE_DIR)/macos/foset $(LDFLAGS) *.go
	mkdir -p $(LATEST_DIR)/macos
	cp $(RELEASE_DIR)/macos/foset $(LATEST_DIR)/macos/foset

linux:
	mkdir -p $(RELEASE_DIR)/linux
	GOOS=linux   GOARCH=amd64 go build -o $(RELEASE_DIR)/linux/foset $(LDFLAGS) *.go
	mkdir -p $(LATEST_DIR)/linux
	cp $(RELEASE_DIR)/linux/foset $(LATEST_DIR)/linux/foset

windows:
	mkdir -p $(RELEASE_DIR)/windows
	GOOS=windows GOARCH=amd64 go build -o $(RELEASE_DIR)/windows/foset.exe $(LDFLAGS) *.go
	mkdir -p $(LATEST_DIR)/windows
	cp $(RELEASE_DIR)/windows/foset.exe $(LATEST_DIR)/windows/foset.exe

local:
	go build -o foset $(LDFLAGS) *.go

clean:
	rm foset_macos foset_linux foset_windows.exe

deb:
	rm -rf debian_package
	mkdir -p debian_package/DEBIAN
	DEBIAN/control.sh >debian_package/DEBIAN/control
	mkdir -p debian_package/usr/bin
	GOOS=linux GOARCH=amd64 go build -o debian_package/usr/bin/foset $(LDFLAGS) *.go

	dpkg-deb --root-owner-group --build debian_package foset-$(MAIN_VERSION).deb

	rm -rf debian_package
