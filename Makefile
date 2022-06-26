.PHONY: setup
setup: ## Download project dependencies
	@go mod download
	@go mod tidy

.PHONY: tests
tests: unit-tests

.PHONY: unit-tests
unit-tests:
	go test github.com/Henrod/task-queue/taskqueue -v -tags=unit

.PHONY: setup/dev
setup/dev:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest

.PHONY: lint
lint:
	golangci-lint run --config .golangci.yml --build-tags unit
