.PHONY: build run once test lint

build:
	go build -o bin/agent ./cmd/agent

run: build
	./bin/agent -config=cmd/agent/main.go

once: build
	./bin/agent -config=cmd/agent/main.go -once

test:
	go test ./...

lint:
	gofmt -w .
	go vet ./...