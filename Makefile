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

$(DST_DIR)/path-playground.wasm: $(SRC_DIR)/wasm/main.go
	mkdir -p $(@D) 
	cd $(SRC_DIR)/wasm; GOOS=js GOARCH=wasm go build -o $(ROOT_DIR)/$@ $$(basename "$<")

$(DST_DIR)/wasm_exec.js: $(WASM_EXEC)
	mkdir -p $(@D) 
	cp $< $@

lint:

.PHONY: clean
clean:
	rm -rf $(DST_DIR)
