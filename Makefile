.PHONY generate:
generate:
	@echo "Generating code with go generate"
	@go generate ./...

.PHONY test:
test: generate
	@echo "Running tests"
	@go test -v ./...
