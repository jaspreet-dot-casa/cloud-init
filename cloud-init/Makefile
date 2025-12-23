.PHONY: generate validate usb clean help

# Default target
help:
	@echo "Cloud-Init Makefile"
	@echo ""
	@echo "Targets:"
	@echo "  make generate    - Generate cloud-init.yaml from template"
	@echo "  make validate    - Validate existing cloud-init.yaml"
	@echo "  make usb         - Create bootable USB (interactive)"
	@echo "  make clean       - Remove generated files"
	@echo ""
	@echo "Prerequisites:"
	@echo "  1. Copy secrets.env.template to secrets.env"
	@echo "  2. Edit secrets.env with your values"
	@echo "  3. Run 'make generate'"

# Generate cloud-init.yaml
generate:
	@./generate.sh

# Validate cloud-init.yaml
validate:
	@./generate.sh --validate

# Create bootable USB
usb: generate
	@./create-usb.sh

# Clean generated files
clean:
	@rm -f cloud-init.yaml
	@echo "Cleaned generated files"
