BINARY     := sproot
CMD        := ./cmd/sproot
BUILD_FLAGS := -ldflags "-X main.version=$(shell git describe --tags --always --dirty 2>/dev/null || echo dev)"

.PHONY: build test lint vet tidy clean install run fmt check release-dry-run release-check e2e

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

# Run end-to-end tests against real sprites using SPRITES_TOKEN.
# Requires SPRITES_TOKEN in env. Authenticates, creates sprites with several
# configs, verifies labels, pushes, and destroys. All sprites are destroyed
# even if a step fails (via cleanup trap).
e2e: build
	@if [ -z "$$SPRITES_TOKEN" ]; then echo "SPRITES_TOKEN is not set"; exit 1; fi
	@echo "--- e2e: authenticating"
	sprite auth setup --token "$$SPRITES_TOKEN"
	@mkdir -p ~/.sproot
	@echo "--- e2e: test 1/3 - local config, 5-phase flat run"
	@{ \
		printf "sproot_config_source: local\nsproot_config_local_path: $$(pwd)/testdata/integration\nsproot_config_path: sproot_local_complex.yaml\ntoken_env: SPRITES_TOKEN\ngh_token_env: GITHUB_TOKEN\n" > ~/.sproot/config.yaml; \
		./sproot new spr-e2e-complex --skip-console --skip-verify --debug || exit 1; \
		./sproot outdated | tee /dev/stderr | grep "current" || exit 1; \
		./sproot push --name spr-e2e-complex --no-checkpoint --skip-verify --debug || exit 1; \
		./sproot outdated | tee /dev/stderr | grep "current" || exit 1; \
	}
	@echo "--- e2e: test 2/3 - local config, multi-target web (extends)"
	@{ \
		printf "sproot_config_source: local\nsproot_config_local_path: $$(pwd)/testdata/integration\nsproot_config_path: sproot_local_targets.yaml\ntoken_env: SPRITES_TOKEN\ngh_token_env: GITHUB_TOKEN\n" > ~/.sproot/config.yaml; \
		./sproot new spr-e2e-target --target web --skip-console --skip-verify --debug || exit 1; \
		./sproot outdated | tee /dev/stderr | grep "current" || exit 1; \
	}
	@echo "--- e2e: test 3/3 - list and status"
	@./sproot list
	@./sproot status --host spr-e2e-complex || true
	@echo "--- e2e: cleanup"
	@./sproot destroy --force spr-e2e-complex || true
	@./sproot destroy --force spr-e2e-target  || true
	@echo "--- e2e: done"
