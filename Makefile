MAIN_VERSION=1.0
FOSET_GIT_COMMIT := $(shell git rev-list -1 HEAD)
FORTISESSION_GIT_COMMIT := $(shell cd ../fortisession/; git rev-list -1 HEAD)
LDFLAGS=-ldflags="-s -w -X main.fosetGitCommit=$(FOSET_GIT_COMMIT) -X main.fortisessionGitCommit=$(FORTISESSION_GIT_COMMIT) -X main.mainVersion=$(MAIN_VERSION)"
RELEASE_DIR=release/$(MAIN_VERSION)

all: macos linux windows

macos:
	mkdir -p $(RELEASE_DIR)/macos
	GOOS=darwin  GOARCH=amd64 go build -o $(RELEASE_DIR)/macos/foset $(LDFLAGS) *.go

linux:
	mkdir -p $(RELEASE_DIR)/linux
	GOOS=linux   GOARCH=amd64 go build -o $(RELEASE_DIR)/linux/foset $(LDFLAGS) *.go

windows:
	mkdir -p $(RELEASE_DIR)/windows
	GOOS=windows GOARCH=amd64 go build -o $(RELEASE_DIR)/windows/foset.exe $(LDFLAGS) *.go

clean:
	rm foset_macos foset_linux foset_windows.exe
