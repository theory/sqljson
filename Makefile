WASM_EXEC := $(shell go env GOROOT)/misc/wasm/wasm_exec.js
DST_DIR := docs/playground
SRC_DIR := src/playground
ROOT_DIR := $(dir $(realpath $(lastword $(MAKEFILE_LIST))))

all: $(DST_DIR)/path-playground.wasm $(DST_DIR)/index.html $(DST_DIR)/playground.css

.PHONY: run
run: all
	python3 -m http.server --directory $(DST_DIR)

$(DST_DIR)/index.html: $(SRC_DIR)/index.html $(SRC_DIR)/wasm/go.mod
	mkdir -p $(@D)
	version=$$(cat $(SRC_DIR)/wasm/go.mod | grep sqljson | awk '{print $$3}'); cat $< | sed -e "s!{{version}}!$${version}!g" > $@

$(DST_DIR)/playground.css: $(SRC_DIR)/playground.css
	mkdir -p $(@D)
	cp $< $@

$(DST_DIR)/path-playground.wasm: $(SRC_DIR)/wasm/main.go $(DST_DIR)/wasm_exec.js
	mkdir -p $(@D)
	cd $(SRC_DIR)/wasm; GOOS=js GOARCH=wasm go build -o $(ROOT_DIR)/$@ $$(basename "$<")

$(DST_DIR)/wasm_exec.js: $(WASM_EXEC)
	mkdir -p $(@D)
	cp $< $@

.PHONY: brew-lint-depends # Install linting tools from Homebrew
brew-lint-depends:
	brew install golangci-lint

.PHONY: debian-lint-depends # Install linting tools on Debian
debian-lint-depends:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/bin v1.62.2

.PHONY: lint # Lint the project
lint: .pre-commit-config.yaml
	@env GOOS=js GOARCH=wasm pre-commit run --show-diff-on-failure --color=always --all-files

.PHONY: tidy # Run go mod tidy
tidy:
	cd src/playground/wasm && go mod tidy

.PHONY: golangci-lint # Run golangci-lint
golangci-lint: .golangci.yaml
	cd src/playground/wasm && GOOS=js GOARCH=wasm golangci-lint run --fix --timeout=5m

.PHONY: clean
clean:
	rm -rf $(DST_DIR)
