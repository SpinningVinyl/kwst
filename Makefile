.DEFAULT_GOAL := build
SCDOC := $(shell command -v scdoc 2> /dev/null)

build: *.go
	go build -o build/kwst -v -ldflags="-X 'main.Version=$$(git describe --always)' -X 'main.BuildTime=$$(date)'"
ifndef SCDOC
	$(error "scdoc not installed, skipping man page generation")
endif
	scdoc < kwst.1.scd > build/kwst.1
	gzip build/kwst.1
install: build
	sudo cp build/kwst /usr/local/bin/kwst
ifeq ($(shell test -e $(./build/kwst.1.gz) && echo -n yes), yes)
	sudo cp build/kwst.1.gz /usr/local/man/man1/kwst.1.gz
endif
