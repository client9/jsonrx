
# test - run all unit tests
test:
	go test ./...

# bench - run local benchmarks
bench:
	go test -benchmem -bench .

# compare - run benchmarks comparing against other libraries 
compare:
	cd benchmarks && $(MAKE)

# cover -generate code coverage report
cover:
	rm -f cover.out
	go test -run='^Test' -coverprofile=cover.out -coverpkg=.
	go tool cover -func=cover.out

# lint - do various linting and cleanups
lint:
	go mod tidy
	gofmt -w -s *.go
	golangci-lint run .

# clean - remove any generated files
clean:
	rm -f cover.out coverage.out
	rm -f tojson	
	rm -f benchmarks/mem.out
	rm -f benchmarks/benchmarks.test
	rm -f tojson.test


