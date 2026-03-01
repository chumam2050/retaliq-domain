# Makefile for building retaliq-domain for various platforms
# Usage example:
#   make all          # build deb, windows exe and darwin binary
#   make deb          # build Debian package via build.sh
#   make windows      # build windows executable
#   make macos        # build macOS executable
#   make clean        # remove built artifacts

BINDIR := $(CURDIR)
PACKAGE := retaliq-domain
VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "0.1.0")
VERSION := $(patsubst v%,%,$(VERSION))

# run Go unit tests
.PHONY: test
test:
	go test ./... -v

GOFLAGS :=

.PHONY: all deb windows macos clean test

all: test deb windows macos
	@mkdir -p dist

deb:
	@echo "Building Debian package version $(VERSION)"
	./debian/build.sh
	@mkdir -p dist
	@mv ../$(PACKAGE)_*.deb dist/ 2>/dev/null || true

windows:
	@echo "Cross-compiling for Windows"
	mkdir -p dist
	GOOS=windows GOARCH=amd64 go build -o dist/$(PACKAGE)-$(VERSION)-windows-amd64.exe ./

macos:
	@echo "Building macOS binary"
	mkdir -p dist
	GOOS=darwin GOARCH=amd64 go build -o dist/$(PACKAGE)-$(VERSION)-darwin-amd64 ./

clean:
	rm -rf dist
	rm -f $(PACKAGE) $(PACKAGE)-*darwin* $(PACKAGE)-*windows*
	rm -f ../$(PACKAGE)_*.deb
