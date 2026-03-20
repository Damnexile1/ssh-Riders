.PHONY: build fmt test run-room run-orchestrator run-gateway

fmt:
	gofmt -w $(shell find . -name '*.go' -not -path './vendor/*')

build:
	go build ./...

test:
	go test ./...

run-room:
	go run ./cmd/room

run-orchestrator:
	go run ./cmd/orchestrator

run-gateway:
	go run ./cmd/gateway
