.PHONY: clean install docs-bootstrap docs docs-serve docs-clean

EXECUTABLE=we
WEPKG=./cmd/we

LAST_TAG := $(shell git describe --abbrev=0 --tags)

VENV=.venv
BUMP=$(VENV)/bin/bumpversion
PIP=$(VENV)/bin/pip
MKDOCS=$(VENV)/bin/mkdocs
DOCS_STAMP=$(VENV)/.docs-deps-stamp
PYTHON?=python3
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
	rm -f we
	rm -rf bin
	rm -rf site

docs-bootstrap: $(DOCS_STAMP)

$(DOCS_STAMP): docs/requirements.txt
	$(PYTHON) -m venv $(VENV)
	$(PIP) install --upgrade pip
	$(PIP) install -r docs/requirements.txt
	touch $(DOCS_STAMP)

docs: docs-bootstrap
	$(MKDOCS) build

docs-serve: docs-bootstrap
	$(MKDOCS) serve

docs-clean:
	rm -rf site

we-example: we
	sh examples/hello_world.sh

test:
	go test `go list ./... | grep -v vendor`

$(BINDIR):
	mkdir -p build


.PHONY: goreleaser
goreleaser:
	go install github.com/goreleaser/goreleaser@latest
