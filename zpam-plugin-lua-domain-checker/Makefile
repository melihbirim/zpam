.PHONY: build test clean validate publish

build:
	@echo "Building lua-domain-checker plugin..."
	@mkdir -p bin
	@go build -o bin/lua-domain-checker src/main.go
	@echo "✅ Build complete"

test:
	@echo "Testing lua-domain-checker plugin..."
	@go test ./src/...
	@echo "✅ Tests passed"

validate:
	@echo "Validating plugin..."
	@zpam plugins validate
	@echo "✅ Validation complete"

clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin/ dist/
	@echo "✅ Clean complete"

publish:
	@echo "Publishing plugin..."
	@zpam plugins build
	@zpam plugins publish
	@echo "✅ Published successfully"
