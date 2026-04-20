
test:
	go test ./...
bench:
	go test -benchmem -bench .

.PHONY: benchmarks
benchmarks:
	cd benchmarks && $(MAKE)
cover:
	rm -f cover.out
	go test -run='^Test' -coverprofile=cover.out -coverpkg=.
	go tool cover -func=cover.out

lint:
	go mod tidy
	gofmt -w -s *.go
	golangci-lint run .

clean:
	rm -f cover.out

	
