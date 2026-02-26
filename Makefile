.PHONY: proto build build-agent build-collector build-simulator run once run-collector run-simulator test lint clean

# ====== Variables ======
AGENT_BIN=bin/agent
COLLECTOR_BIN=bin/collector
SIMULATOR_BIN=bin/simulator
PROTO_DIR=proto

# ====== Proto ======
proto:
	protoc \
		--go_out=. \
		--go-grpc_out=. \
		$(PROTO_DIR)/*.proto

# ====== Build ======
build-agent: proto
	go build -o $(AGENT_BIN) ./cmd/agent

build-collector: proto
	go build -o $(COLLECTOR_BIN) ./cmd/collector

build-simulator: proto
	go build -o $(SIMULATOR_BIN) ./cmd/simulator

# ====== Run ======
run: build-agent
	./$(AGENT_BIN) -config=./config.json

once: build-agent
	./$(AGENT_BIN) -config=./config.json -once

run-collector: build-collector
	./$(COLLECTOR_BIN) -listen=:50051

run-simulator: build-simulator
	./$(SIMULATOR_BIN) -agents=5 -scenario=full -duration=30s

# ====== Dev ======
test:
	go test ./...

lint:
	gofmt -w .
	go vet ./...

clean:
	rm -rf bin