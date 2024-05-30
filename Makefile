SHELL:=/usr/bin/env bash

BIN_NAME:=xtool
BIN_VERSION:=$(shell ./.version.sh)
X3F_EXTRACT_VERSION:=$(shell ./x3f_extract 2>/dev/null | grep -i "VERSION =" | rev | cut -d' ' -f1 | rev)

default: help
.PHONY: help
help: ## Display this help message
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

applescript-embed.tar: $(shell find ./applescript -type f -name '*.scpt' | sed 's/ /\\ /g') \
		$(shell find ./applescript -type f -name '*.rsrc' | sed 's/ /\\ /g') \
		./applescript/gen-archive.sh \
		./applescript/install.sh \
		./applescript/restore-resources.sh
	./applescript/gen-archive.sh

.PHONY: deps
deps: applescript-embed.tar

.PHONY: clean
clean: ## Remove build products
	rm -f applescript-embed.tar
	rm -rf ./out

.PHONY: fmt
fmt: applescript-embed.tar ## Run automatic code formatters on the codebase
	go fmt *.go
	prettier --write .
	shfmt -l -w .

.PHONY: lint
lint: applescript-embed.tar ## Lint the codebase
	golangci-lint run
	shellcheck ./applescript/*.sh
	prettier --check .
	shfmt -d .
	actionlint .github/workflows/*.yml

.PHONY: build
build: applescript-embed.tar ## Build (for macOS/arm64 and macOS/amd64) to ./out
	mkdir -p out
	env CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 go build -ldflags="-X main.Version=${BIN_VERSION} -X main.X3fExtractVersion=${X3F_EXTRACT_VERSION}" -o ./out/${BIN_NAME}-${BIN_VERSION}-darwin-arm64 .
	env CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 go build -ldflags="-X main.Version=${BIN_VERSION} -X main.X3fExtractVersion=${X3F_EXTRACT_VERSION}" -o ./out/${BIN_NAME}-${BIN_VERSION}-darwin-amd64 .
