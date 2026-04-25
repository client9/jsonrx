SHELL := sh

.PHONY: help
.DEFAULT_GOAL := help
help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}' 

build: ## build module and CLI
	go build ./...
	go build ./cmd/tojson/...

test: ## run all unit tests
	go test ./...

test-toml: ## run only TOML tests
	go test -v -run TOML ./...

version: ## print OS, Go, and golangci versions
	@echo $$0
	@uname -a
	@go version
	@golangci-lint --version

bench: ## run local benchmarks
	go test -benchmem -bench .

compare: ## run benchmarks comparing against other libraries
	cd benchmarks && $(MAKE)

cover: ## generate code coverage report
	rm -f cover.out
	go test -run='^Test' -coverprofile=cover.out -coverpkg=.
	go tool cover -func=cover.out

## NOTE: this downloads it's schema over the network
lintverify:
	golangci-lint config verify

fmt: ## reformat source code
	go mod tidy
	gofmt -w -s *.go

lint: ## lint and verify repo is already formatted
	go mod tidy
	git diff --exit-code -- go.mod go.sum
	test -z "$$(gofmt -l *.go)"
	golangci-lint run .

clean: ## remove any generated files
	rm -f *.out benchmarks/*.out
	rm -f tojson	
	rm -f benchmarks/mem.out
	rm -f benchmarks/benchmarks.test
	rm -f tojson.test


