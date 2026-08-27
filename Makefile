BINARY_NAME=waste-dispatch
BUILD_DIR=./bin
CMD_PATH=./cmd/server

.PHONY: all build run test clean lint tidy docker-build migrate-up migrate-down

## build: Compile the binary
build:
	@mkdir -p $(BUILD_DIR)
	go build -ldflags="-s -w" -o $(BUILD_DIR)/$(BINARY_NAME) $(CMD_PATH)

## run: Run the server locally
run:
	go run $(CMD_PATH)/main.go

## test: Run all tests
test:
	go test -v -race -count=1 ./...

## test-cover: Run tests with coverage report
test-cover:
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

## lint: Run golangci-lint
lint:
	golangci-lint run ./...

## tidy: Tidy and verify go modules
tidy:
	go mod tidy
	go mod verify

## clean: Remove build artifacts
clean:
	rm -rf $(BUILD_DIR) coverage.out coverage.html data/

## docker-build: Build the Docker image
docker-build:
	docker build -t $(BINARY_NAME):latest .

## docker-run: Run the Docker container
docker-run:
	docker run --rm -p 8080:8080 \
		-e AUTH_TOKEN_SECRET=dev-secret-change-in-prod \
		-v $(PWD)/data:/app/data \
		$(BINARY_NAME):latest

## migrate-up: Apply database migrations (requires migrate CLI)
migrate-up:
	migrate -path ./migrations -database "sqlite3://./data/waste_dispatch.db" up

## migrate-down: Roll back all migrations (requires migrate CLI)
migrate-down:
	migrate -path ./migrations -database "sqlite3://./data/waste_dispatch.db" down

## help: Print this help message
help:
	@echo "Available targets:"
	@sed -n 's/^## //p' $(MAKEFILE_LIST) | column -t -s ':'
