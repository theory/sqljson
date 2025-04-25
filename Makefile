GO ?= go

.PHONY: test # Run the unit tests
test:
	GOTOOLCHAIN=local $(GO) test ./... -count=1

.PHONY: cover # Run test coverage
cover: $(shell find . -name \*.go)
	GOTOOLCHAIN=local $(GO) test -v -coverprofile=cover.out -covermode=count ./...
	@$(GO) tool cover -html=cover.out

.PHONY: lint # Lint the project
lint: .golangci.yaml
	@pre-commit run --show-diff-on-failure --color=always --all-files

.PHONY: clean # Remove generated files
clean:
	$(GO) clean
	@rm -rf cover.out

############################################################################
# Utilities.
.PHONY: brew-lint-depends # Install linting tools from Homebrew
brew-lint-depends:
	brew install golangci-lint

.PHONY: debian-lint-depends # Install linting tools on Debian
debian-lint-depends:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sudo sh -s -- -b /usr/bin v2.1.5

.PHONY: install-generators # Install Go code generators
install-generators:
	$(GO) install golang.org/x/tools/cmd/goyacc@v0.32.0
	$(GO) install golang.org/x/tools/cmd/stringer@v0.32.0

.PHONY: generate # Generate Go code
generate:
	@$(GO) generate ./...
	@perl -i -pe 's{^//line yacc.+\n}{}g' path/parser/grammar.go

## .git/hooks/pre-commit: Install the pre-commit hook
.git/hooks/pre-commit:
	@printf "#!/bin/sh\nmake lint\n" > $@
	@chmod +x $@

.PHONY: pg-diff # Generage diff statements aginst the Postgres source.
pg-diff: .util/pglist.go
	@go run $<
