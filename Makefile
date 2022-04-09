.DEFAULT_GOAL := all

.PHONY: all
all: lint test container_scheduler

.PHONY: test
test: generate
	go test -race -timeout 10s -count 1 ./...

.PHONY: generate
generate:
	go install github.com/gojuno/minimock/v3/cmd/minimock@latest
	go generate ./...

.PHONY: lint
lint:
	golangci-lint --timeout=5m run

.PHONY: container_scheduler
container_scheduler:
	go build -o ./bin/container_scheduler ./cmd
