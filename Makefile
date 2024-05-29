SHELL:=/usr/bin/env bash
VERSION:=$(shell [ -z "$$(git tag --points-at HEAD)" ] && echo "$$(git describe --always --long --dirty | sed 's/^v//')" || echo "$$(git tag --points-at HEAD | sed 's/^v//')")
X3F_EXTRACT_VERSION:=$(shell ./x3f_extract 2>/dev/null | grep -i "VERSION =" | rev | cut -d' ' -f1 | rev)
BIN_NAME:=xtool

default: help

# via https://marmelab.com/blog/2016/02/29/auto-documented-makefile.html
.PHONY: help
help: ## Print help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

.PHONY: clean
clean: ## Remove built products in ./out
	rm -rf ./out

.PHONY: lint
lint: ## Lint all .go files
	golangci-lint run

.PHONY: build
build: lint ## Build (for the current platform & architecture) to ./out
	mkdir -p out
	go build -ldflags="-X main.version=${VERSION} -X main.X3fExtractVersion=${X3F_EXTRACT_VERSION}" -o ./out/${BIN_NAME} .

.PHONY: install
install: ## Build & install xtool to /usr/local/bin, without linting
	go build -ldflags="-X main.version=${VERSION}" -o /usr/local/bin/${BIN_NAME} .

.PHONY: applescript-install
applescript-install: ## Copy supporting AppleScripts to ~/Library/Scripts/Applications/Finder
	./applescript/restore-resources.sh
	cp -vf "applescript/xtool - "*.scpt "$$HOME/Library/Scripts/Applications/Finder"
