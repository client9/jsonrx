
.PHONY: help

help:
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-30s\033[0m %s\n", $$1, $$2}'

build: ## build module and CLI
	go build ./...
	go build ./cmd/tojson/...

test: ## run all unit tests
	go test ./...

version: ## print OS, Go, and golangci versions
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

lint: ## do various linting and cleanups
	go mod tidy
	gofmt -w -s *.go
	golangci-lint config verify
	golangci-lint run .

clean: ## remove any generated files
	rm -f *.out benchmarks/*.out
	rm -f tojson	
	rm -f benchmarks/mem.out
	rm -f benchmarks/benchmarks.test
	rm -f tojson.test


