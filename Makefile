.DEFAULT_GOAL := build
SCDOC := $(shell command -v scdoc 2> /dev/null)
SCD2HTML := $(shell command -v scd2html 2> /dev/null)
PREFIX ?= /usr/local
DESTDIR ?=
BUILD_DIR := build

.PHONY: integration-test

build: *.go *.scd
	mkdir -p $(BUILD_DIR)
	go build -a -o $(BUILD_DIR)/kwst -v -ldflags="-X 'main.Version=$$(git describe --always)' -X 'main.BuildTime=$$(date)'"
ifdef SCDOC
	scdoc < kwst.1.scd > $(BUILD_DIR)/kwst.1
	gzip -f $(BUILD_DIR)/kwst.1
else
	$(info "scdoc not installed, skipping man page generation")
endif
ifdef SCD2HTML
	scd2html < kwst.1.scd > usage.html
else
	$(info "scd2html not installed, skipping HTML documentation generation")
endif

install: build
	install -d "$(DESTDIR)$(PREFIX)/bin"
	install -m 0755 $(BUILD_DIR)/kwst "$(DESTDIR)$(PREFIX)/bin/kwst"
	if [ -f $(BUILD_DIR)/kwst.1.gz ]; then \
		install -d "$(DESTDIR)$(PREFIX)/share/man/man1"; \
		install -m 0644 $(BUILD_DIR)/kwst.1.gz "$(DESTDIR)$(PREFIX)/share/man/man1/kwst.1.gz"; \
	fi

integration-test:
	KWST_INTEGRATION=1 go test -tags=integration -count=1 -timeout=2m ./integration
