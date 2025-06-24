
test:
	go test ./...
bench:
	go test -benchmem -bench .

lint:
	go mod tidy
	gofmt -w -s *.go
	golangci-lint run .
