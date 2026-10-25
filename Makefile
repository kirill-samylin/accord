GO ?= $(if $(wildcard .local/toolchains/go/bin/go),$(CURDIR)/.local/toolchains/go/bin/go,go)
GOFMT ?= $(if $(wildcard .local/toolchains/go/bin/gofmt),$(CURDIR)/.local/toolchains/go/bin/gofmt,gofmt)
export GOTOOLCHAIN := local
export GOCACHE := $(CURDIR)/.local/toolchains/cache/build
export GOMODCACHE := $(CURDIR)/.local/toolchains/cache/mod
export GOTMPDIR := $(CURDIR)/.local/toolchains/tmp
export NEXT_TELEMETRY_DISABLED := 1

PYTHON ?= python3
NODE ?= node
NEXT_NODE ?= $(NODE)
NPM ?= npm
WEB_PORT ?= 3100

.PHONY: build test infra-test race vet fmt-check check web-install web-format-check web-typecheck web-test web-build web-check web-dev web-start

build:
	@mkdir -p bin "$(GOTMPDIR)"
	"$(GO)" build -o bin/agent-board ./cmd/agent-board

test:
	@mkdir -p "$(GOTMPDIR)"
	"$(GO)" test ./...

infra-test:
	PYTHONDONTWRITEBYTECODE=1 "$(PYTHON)" -m unittest discover -s tests/infra -p 'test_*.py' -v

web-install:
	"$(NPM)" --prefix web ci --registry=https://registry.npmjs.org

web-format-check:
	cd web && "$(NEXT_NODE)" node_modules/prettier/bin/prettier.cjs --check src next.config.ts postcss.config.mjs tsconfig.json package.json

web-typecheck:
	cd web && "$(NEXT_NODE)" node_modules/typescript/bin/tsc --noEmit

web-test:
	cd web && "$(NEXT_NODE)" --experimental-strip-types --test src/lib/*.test.ts src/features/mutations/*.test.ts

web-build:
	cd web && "$(NEXT_NODE)" node_modules/next/dist/bin/next build

web-check: web-format-check web-typecheck web-test web-build

web-dev:
	cd web && "$(NEXT_NODE)" node_modules/next/dist/bin/next dev --hostname 127.0.0.1 --port $(WEB_PORT)

web-start:
	cd web && "$(NEXT_NODE)" node_modules/next/dist/bin/next start --hostname 127.0.0.1 --port $(WEB_PORT)

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

check: fmt-check vet test infra-test race build web-check
