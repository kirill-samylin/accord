GO ?= $(if $(wildcard .local/toolchains/go/bin/go),$(CURDIR)/.local/toolchains/go/bin/go,go)
GOFMT ?= $(if $(wildcard .local/toolchains/go/bin/gofmt),$(CURDIR)/.local/toolchains/go/bin/gofmt,gofmt)
export GOTOOLCHAIN := local
export GOCACHE := $(CURDIR)/.local/toolchains/cache/build
export GOMODCACHE := $(CURDIR)/.local/toolchains/cache/mod
export GOTMPDIR := $(CURDIR)/.local/toolchains/tmp

.PHONY: build test race vet fmt-check check

build:
	@mkdir -p bin "$(GOTMPDIR)"
	"$(GO)" build -o bin/agent-board ./cmd/agent-board

test:
	@mkdir -p "$(GOTMPDIR)"
	"$(GO)" test ./...

race:
	@mkdir -p "$(GOTMPDIR)"
	"$(GO)" test -race ./...

vet:
	@mkdir -p "$(GOTMPDIR)"
	"$(GO)" vet ./...

fmt-check:
	@files=$$(find cmd internal tests -name '*.go' -type f 2>/dev/null); \
	if [ -n "$$files" ]; then \
		unformatted=$$("$(GOFMT)" -l $$files) || exit $$?; \
		if [ -n "$$unformatted" ]; then printf '%s\n' "$$unformatted"; exit 1; fi; \
	fi

check: fmt-check vet test race build
