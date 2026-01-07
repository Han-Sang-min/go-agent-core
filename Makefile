.PHONY: proto build build-agent build-collector run once run-collector test lint clean

# ====== Variables ======
AGENT_BIN=bin/agent
COLLECTOR_BIN=bin/collector
PROTO_DIR=proto

# ====== Proto ======
proto:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		$(PROTO_DIR)/*.proto

# ====== Build ======
build: build-agent
	go build -o bin/agent ./cmd/agent

build-agent: proto
	go build -o $(AGENT_BIN) ./cmd/agent

build-collector: proto
	go build -o $(COLLECTOR_BIN) ./cmd/collector

# ====== Run ======
run: build-agent
	./$(AGENT_BIN) -config=./config.json

once: build-agent
	./$(AGENT_BIN) -config=./config.json -once

run-collector: build-collector
	./$(COLLECTOR_BIN) -listen=:50051

# ====== Dev ======
test:
	go test ./...

lint:
	gofmt -w .
	go vet ./...

clean:
	rm -rf bin