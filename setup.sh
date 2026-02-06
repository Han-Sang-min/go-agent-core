#!/bin/bash
set -e

echo ">>> [1/4] Installing System Dependencies..."
sudo apt-get update
sudo apt-get install -y make curl protobuf-compiler

echo ">>> [2/4] Installing Kind..."
if ! command -v kind &> /dev/null; then
    curl -Lo ./kind https://kind.sigs.k8s.io/dl/v0.20.0/kind-linux-amd64
    chmod +x ./kind
    sudo mv ./kind /usr/local/bin/kind
else
    echo "Kind is already installed."
fi

echo ">>> [3/4] Creating Kubernetes Cluster..."
if ! kind get clusters | grep -q "k8s-learn"; then
    kind create cluster --name k8s-learn
else
    echo "Cluster 'k8s-learn' already exists."
fi

echo ">>> [4/4] Setting up Go Tools..."
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest
export PATH="$PATH:$(go env GOPATH)/bin"

echo "protoc=$(protoc --version)"
echo "protoc-gen-go=$(protoc-gen-go --version)"
echo "protoc-gen-go-grpc=$(protoc-gen-go-grpc --version)"

make proto
go mod tidy

echo ">>> Done!"