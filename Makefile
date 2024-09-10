default: test


.PHONY: test
test:
	@echo "Running tests..."
	@go test -v ./...
	@echo "All tests passed!"

.PHONY: clean
clean:
	@echo "Cleaning up..."
	@rm -rf ./bin
	@rm -rf ./coverage

.PHONY: build
build:
	@echo "Building..."
	@go build -o ./bin/wmb ./cmd/
