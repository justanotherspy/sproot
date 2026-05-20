BINARY     := sproot
CMD        := ./cmd/sproot
BUILD_FLAGS := -ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)"

.PHONY: build test lint vet tidy clean install run fmt check release-dry-run release-check

build:
	go build $(BUILD_FLAGS) -o $(BINARY) $(CMD)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-race:
	go test -race ./...

GOLANGCI_LINT ?= $(shell go env GOPATH)/bin/golangci-lint

lint:
	$(GOLANGCI_LINT) run ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

clean:
	rm -f $(BINARY)

install:
	go install $(BUILD_FLAGS) $(CMD)

run:
	go run $(BUILD_FLAGS) $(CMD)

# Run all checks locally before pushing
check: vet test lint

# Dry-run the release pipeline locally (requires goreleaser in PATH).
# Skips signing (needs GitHub OIDC) and publishing.
release-dry-run:
	goreleaser release --snapshot --clean --skip=publish,sign

# Validate .goreleaser.yaml syntax without building.
release-check:
	goreleaser check
