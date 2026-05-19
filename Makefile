BINARY     := sproot
CMD        := ./cmd/sproot
BUILD_FLAGS := -ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)"

.PHONY: build test lint vet tidy clean install run fmt check

build:
	go build $(BUILD_FLAGS) -o $(BINARY) $(CMD)

test:
	go test ./...

test-verbose:
	go test -v ./...

test-race:
	go test -race ./...

lint:
	golangci-lint run ./...

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
