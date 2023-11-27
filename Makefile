.PHONY: clean install

EXECUTABLE=we
WEPKG=./cmd/we

LAST_TAG := $(shell git describe --abbrev=0 --tags)

VENV=.venv
BUMP=$(VENV)/bin/bumpversion
BUMPTYPE=patch

SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -path ./docker -prune -o -name '*.go')

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

VERSION=`git describe --tags --long --always --dirty`
LDFLAGS=-ldflags "-X main.builddate=$(BUILD_TIME) -X main.gitref=$(VERSION)"

we: $(SOURCES) tidy
	go build $(LDFLAGS) -o $(EXECUTABLE) $(WEPKG)

install: tidy
	go install $(WEPKG)

tidy:
	go mod tidy

clean:
	rm we
	rm -rf bin

we-example: we
	sh examples/hello_world.sh

test:
	go test `go list ./... | grep -v vendor`

$(BINDIR):
	mkdir -p build


.PHONY: goreleaser
goreleaser:
	go install github.com/goreleaser/goreleaser@latest
