.PHONY: build run once test lint

build:
	go build -o bin/agent ./cmd/agent

run: build
	./bin/agent -config=./config.json

once: build
	./bin/agent -config=./config.json -once

test:
	go test ./...

lint:
	gofmt -w .
	go vet ./...