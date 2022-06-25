.PHONY: tests
tests: unit-tests

.PHONY: unit-tests
unit-tests:
	go test github.com/Henrod/task-queue/taskqueue -v -tags=unit

.PHONY: integration-tests
integration-tests:
	go test -timeout 5s -tags unit github.com/Henrod/task-queue/test

.PHONY: setup
setup:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml --build-tags unit
