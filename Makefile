.DEFAULT_GOAL := build
SCDOC := $(shell command -v scdoc 2> /dev/null)

build: *.go
	go build -o build/kwst -v -ldflags="-X 'main.Version=$$(git describe --always)' -X 'main.BuildTime=$$(date)'"
ifdef SCDOC
	scdoc < kwst.1.scd > build/kwst.1
	gzip -f build/kwst.1
else
	$(info "scdoc not installed, skipping man page generation")
endif
install:
	cp build/kwst /usr/local/bin/kwst
ifeq ($(shell test -e $(./build/kwst.1.gz) && echo -n ok), ok)
	cp build/kwst.1.gz /usr/local/man/man1/kwst.1.gz
endif
