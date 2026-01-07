.PHONY: help update update-dry verify-cloud test-syntax lint shellcheck test-multipass test-multipass-keep test-multipass-clean pre-commit build-cli run-cli test-cli install-cli clean-cli generate-templ

# Get the directory where this Makefile resides
ROOT := $(dir $(abspath $(lastword $(MAKEFILE_LIST))))

help:
	echo "Available targets:"
	echo ""
	echo "Go CLI (ucli):"
	echo "  make build-cli         - Build the ucli binary"
	echo "  make run-cli           - Run the CLI interactively"
	echo "  make test-cli          - Run Go tests"
	echo "  make install-cli       - Install ucli to GOPATH/bin"
	echo "  make clean-cli         - Clean build artifacts"
	echo "  make generate-templ    - Generate Go code from templ files"
	echo ""
	echo "Cloud-Init Installation:"
	echo "  make update            - Update all packages idempotently"
	echo "  make update-dry        - Preview changes without applying"
	echo "  make verify-cloud      - Verify cloud-init installation"
	echo ""
	echo "Testing & Validation:"
	echo "  make pre-commit        - Quick tests before committing (syntax + unit tests)"
	echo "  make test-multipass      - Run cloud-init test in Multipass VM"
	echo "  make test-multipass-keep - Run test but keep VM for debugging"
	echo "  make test-multipass-clean- Clean up leftover test VMs"
	echo "  make test-syntax         - Check bash/zsh syntax of all scripts"
	echo "  make lint                - Run shellcheck linter on all scripts"

# Pre-commit checks - fast tests before committing
pre-commit:
	@echo "════════════════════════════════════════════"
	@echo "Running pre-commit checks..."
	@echo "════════════════════════════════════════════"
	@echo ""
	@$(MAKE) test-syntax
	@echo ""
	@echo "Running unit tests..."
	@bash $(ROOT)tests/test-runner.sh
	@echo ""
	@echo "════════════════════════════════════════════"
	@echo "✓ All pre-commit checks passed!"
	@echo "════════════════════════════════════════════"

test-syntax:
	echo "Checking script syntax..."
	echo "Checking shared scripts..."
	for f in $(ROOT)scripts/shared/*.sh; do bash -n "$$f" && echo "  ✓ shared/$$(basename $$f)"; done
	echo "Checking library scripts..."
	for f in $(ROOT)scripts/lib/*.sh; do bash -n "$$f" && echo "  ✓ lib/$$(basename $$f)"; done
	echo "Checking package scripts..."
	for f in $(ROOT)scripts/packages/*.sh; do bash -n "$$f" && echo "  ✓ packages/$$(basename $$f)"; done
	echo "Checking cloud-init scripts..."
	for f in $(ROOT)scripts/cloud-init/*.sh; do bash -n "$$f" && echo "  ✓ cloud-init/$$(basename $$f)"; done
	echo ""
	echo "✓ All scripts have valid syntax"

lint:
	@echo "Running shellcheck on all scripts..."
	shellcheck $(ROOT)scripts/lib/*.sh
	@echo "  ✓ scripts/lib/"
	shellcheck $(ROOT)scripts/packages/*.sh
	@echo "  ✓ scripts/packages/"
	shellcheck $(ROOT)scripts/cloud-init/*.sh
	@echo "  ✓ scripts/cloud-init/"
	shellcheck $(ROOT)scripts/shared/*.sh
	@echo "  ✓ scripts/shared/"
	shellcheck $(ROOT)cloud-init/*.sh
	@echo "  ✓ cloud-init/"
	@echo ""
	@echo "✓ All scripts passed shellcheck"

# Alias for backwards compatibility
shellcheck: lint

# ============================================================================
# Cloud-Init Targets
# ============================================================================

# Update all packages idempotently (for cloud-init systems)
update:
	if [ -f $(ROOT)scripts/cloud-init/update-all.sh ]; then \
		bash $(ROOT)scripts/cloud-init/update-all.sh; \
	else \
		echo "Error: Cloud-init update script not found"; \
		echo "Expected: scripts/cloud-init/update-all.sh"; \
		exit 1; \
	fi

# Dry-run mode - preview changes without applying
update-dry:
	if [ -f $(ROOT)scripts/cloud-init/update-all.sh ]; then \
		DRY_RUN=true bash $(ROOT)scripts/cloud-init/update-all.sh; \
	else \
		echo "Cloud-init scripts not yet implemented"; \
		exit 1; \
	fi

# Verify cloud-init installation
verify-cloud:
	if [ -f $(ROOT)scripts/cloud-init/update-all.sh ]; then \
		bash $(ROOT)scripts/cloud-init/update-all.sh --verify-only; \
	else \
		echo "Cloud-init scripts not yet implemented"; \
		exit 1; \
	fi

# ============================================================================
# Multipass VM Testing
# ============================================================================

# Run full cloud-init test in Multipass VM
test-multipass:
	echo "Running cloud-init integration test in Multipass VM..."
	bash $(ROOT)tests/multipass/run-test.sh

# Run test but keep VM for debugging
test-multipass-keep:
	echo "Running cloud-init test (keeping VM for debugging)..."
	bash $(ROOT)tests/multipass/run-test.sh --keep

# Clean up any leftover test VMs
test-multipass-clean:
	echo "Cleaning up test VMs..."
	multipass list --format csv 2>/dev/null | grep cloud-init-test | cut -d',' -f1 | xargs -I {} multipass delete {} 2>/dev/null || true
	multipass purge 2>/dev/null || true
	echo "Cleanup complete"

# ============================================================================
# Go CLI Targets
# ============================================================================

# Version injection via ldflags
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -ldflags "-X main.version=$(VERSION)"

# Generate templ files
generate-templ:
	@echo "Generating templ files..."
	go tool templ generate
	@echo "✓ Generated templ files"

# Build the ucli binary
build-cli: generate-templ
	@echo "Building ucli (version: $(VERSION))..."
	go build $(LDFLAGS) -o bin/ucli ./cmd/ucli
	@echo "✓ Built: bin/ucli"

# Run the CLI interactively
run-cli: build-cli
	./bin/ucli create

# Run Go tests
test-cli:
	@echo "Running Go tests..."
	go test -v ./...

# Install CLI to GOPATH/bin
install-cli:
	@echo "Installing ucli..."
	go install ./cmd/ucli
	@echo "✓ Installed: ucli"

# Clean build artifacts
clean-cli:
	rm -rf bin/ucli
	@echo "✓ Cleaned: bin/ucli"
