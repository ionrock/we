.PHONY: clean install

SOURCEDIR=.
SOURCES := $(shell find $(SOURCEDIR) -path ./docker -prune -o -name '*.go')

VERSION=1.0.0
BUILD_TIME=`date +%FT%T%z`

VERSION=`git describe --tags --long --always --dirty`
LDFLAGS=-ldflags "-X main.builddate=$(BUILD_TIME) -X main.gitref=$(VERSION)"

GLIDE=$(GOPATH)/bin/glide

we: $(SOURCES) $(GLIDE)
	go build -o we $(LDFLAGS) .

install: $(GLIDE)
	go install ./cmd/...

$(GLIDE):
	go get github.com/Masterminds/glide
	glide i

clean:
	rm we

we-example: we
	./we -e example_env.yml echo 'Hello World!'

test:
	go test

build-all: $(GLIDE) $(SOURCES)
	glide i

	# amd64
	GOOS=linux GOARCH=amd64 go build -o $(BINDIR)/we-linux-amd64       ${LDFLAGS} ./cmd/we/
	GOOS=windows GOARCH=amd64 go build -o $(BINDIR)/we-windows-amd64       ${LDFLAGS} ./cmd/we/
	GOOS=darwin GOARCH=amd64 go build -o $(BINDIR)/we-darwin-amd64       ${LDFLAGS} ./cmd/we/

	# i386
	GOOS=linux GOARCH=386 go build -o $(BINDIR)/we-linux-386       ${LDFLAGS} ./cmd/we/
	GOOS=windows GOARCH=386 go build -o $(BINDIR)/we-windows-386       ${LDFLAGS} ./cmd/we/
	GOOS=darwin GOARCH=386 go build -o $(BINDIR)/we-darwin-386       ${LDFLAGS} ./cmd/we/
