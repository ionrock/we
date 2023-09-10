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


### Release / deploy targets
GH_RELEASE=$(GOPATH)/bin/github-release
UPLOAD_CMD = ./we -e .release.yml $(GH_RELEASE) upload -t $(LAST_TAG) -n $(subst /,-,$(FILE)) -f $(FILE)

OSLIST = darwin freebsd linux
PLATS  = amd64 386

EXE_TMPL     = bin/$(OS)-$(PLAT)-$(EXECUTABLE)
EXE_TMPL_BZ2 = $(FN).tar.bz2

UNIX_EXECUTABLES = $(foreach OS,$(OSLIST),$(foreach PLAT,$(PLATS),$(EXE_TMPL)))
WIN_EXECUTABLES  = bin/windows-amd64-$(EXECUTABLE).exe

COMPRESSED_EXECUTABLES = $(UNIX_EXECUTABLES:%=%.tar.bz2) $(WIN_EXECUTABLES).zip
COMPRESSED_EXECUTABLE_TARGETS = $(COMPRESSED_EXECUTABLES:%=%)

all: $(UNIX_EXECUTABLES) $(WIN_EXECUTABLES)

compress-all: $(COMPRESSED_EXECUTABLES)

# compressed artifacts, makes a huge difference (Go executable is ~9MB,
# after compressing ~2MB)
%.tar.bz2: %
	tar -jcvf "$<.tar.bz2" "$<"
%.exe.zip: %.exe
	zip "$@" "$<"

# 386
bin/darwin-386-$(EXECUTABLE): $(GLIDE) $(SOURCES)
	GOARCH=386 GOOS=darwin go build $(LDFLAGS) -o "$@" $(WEPKG)
bin/linux-386-$(EXECUTABLE): $(GLIDE) $(SOURCES)
	GOARCH=386 GOOS=linux go build $(LDFLAGS) -o "$@" $(WEPKG)
bin/freebsd-386-$(EXECUTABLE): $(GLIDE) $(SOURCES)
	GOARCH=386 GOOS=freebsd go build $(LDFLAGS) -o "$@" $(WEPKG)

bin/windows-386-$(EXECUTABLE).exe: $(GLIDE) $(SOURCES)
	GOARCH=386 GOOS=windows go build $(LDFLAGS) -o "$@" $(WEPKG)

# amd64
bin/freebsd-amd64-$(EXECUTABLE): $(GLIDE) $(SOURCES)
	GOARCH=amd64 GOOS=freebsd go build $(LDFLAGS) -o "$@" $(WEPKG)
bin/darwin-amd64-$(EXECUTABLE): $(GLIDE) $(SOURCES)
	GOARCH=amd64 GOOS=darwin go build $(LDFLAGS) -o "$@" $(WEPKG)
bin/linux-amd64-$(EXECUTABLE): $(GLIDE) $(SOURCES)
	GOARCH=amd64 GOOS=linux go build $(LDFLAGS) -o "$@" $(WEPKG)

bin/windows-amd64-$(EXECUTABLE).exe: $(GLIDE) $(SOURCES)
	GOARCH=amd64 GOOS=windows go build $(LDFLAGS) -o "$@" $(WEPKG)


$(GH_RELEASE):
	go get github.com/aktau/github-release
	go install github.com/aktau/github-release

release: $(GH_RELEASE) $(EXECUTABLE) $(COMPRESSED_EXECUTABLES)
	@echo "All targets $(COMPRESSED_EXECUTABLES)"
	./we -e .release.yml $(GH_RELEASE) release -t $(LAST_TAG) -n $(LAST_TAG) || true
	$(foreach FILE,$(COMPRESSED_EXECUTABLES),$(UPLOAD_CMD);)

$(BUMP):
	virtualenv $(VENV)
	$(VENV)/bin/pip install --upgrade pip
	$(VENV)/bin/pip install bumpversion

bump: $(BUMP)
	$(VENV)/bin/bumpversion $(BUMPTYPE)
	git push && git push --tags
