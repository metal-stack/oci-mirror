
SHA := $(shell git rev-parse --short=8 HEAD)
GITVERSION := $(shell git describe --long --all)
BUILDDATE := $(shell date -Iseconds)
VERSION := $(or ${VERSION},$(shell git describe --tags --exact-match 2> /dev/null || git symbolic-ref -q --short HEAD || git rev-parse --short HEAD))

CGO_ENABLED := 1
LINKMODE := -extldflags '-static -s -w'

all: test server

.PHONY: server
server:
	go build -tags netgo,osusergo,urfave_cli_no_docs \
		 -ldflags "$(LINKMODE) -X 'github.com/metal-stack/v.Version=$(VERSION)' \
								   -X 'github.com/metal-stack/v.Revision=$(GITVERSION)' \
								   -X 'github.com/metal-stack/v.GitSHA1=$(SHA)' \
								   -X 'github.com/metal-stack/v.BuildDate=$(BUILDDATE)'" \
	   -o bin/server github.com/metal-stack/oci-mirror/cmd
	strip bin/server

.PHONY: test
test:
	go test ./... -race -coverprofile=coverage.out -covermode=atomic && go tool cover -func=coverage.out