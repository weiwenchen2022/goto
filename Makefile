.DEFAULT_GOAL := build

.PHONY: build test vet fmt lint

build: vet
	go build

test: vet
	go test -v ./...

vet: fmt
	go vet ./...
	shadow ./...

fmt:
	go fmt ./...

lint: fmt
	golangci-lint run
	# golint ./...
