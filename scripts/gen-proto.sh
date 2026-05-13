#!/bin/bash
set -e

export PATH="$PATH:$(go env GOPATH)/bin"

# Generate Go bindings for proto files
echo "Generating protobuf bindings..."
mkdir -p proto/gen
protoc --go_out=proto/gen --go_opt=paths=source_relative \
       -I=proto proto/events.proto
echo "Protobuf generation complete."
