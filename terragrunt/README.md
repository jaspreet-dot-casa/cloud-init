# Terragrunt + OpenTofu VM Configuration

This directory contains the Terragrunt/OpenTofu configuration for managing libvirt VMs.

## Structure

```text
terragrunt/
├── modules/
│   └── libvirt-vm/          # Reusable VM module
│       ├── main.tf          # VM resources (volume, cloud-init, domain)
│       ├── variables.tf     # Input variables
│       └── outputs.tf       # Output values
├── terragrunt.hcl           # Root config (provider, backend)
└── README.md                # This file

tf/
└── <vm-name>/               # Generated per-VM configs
    ├── terragrunt.hcl       # VM-specific inputs
    └── cloud-init.yaml      # Cloud-init user-data
```

## Prerequisites

1. **OpenTofu** (or Terraform):
   ```bash
   # macOS
   brew install opentofu

   # Linux
   curl -fsSL https://get.opentofu.org/install-opentofu.sh | sh
   ```

2. **Terragrunt**:
   ```bash
   # macOS
   brew install terragrunt

   # Linux
   curl -sL https://github.com/gruntwork-io/terragrunt/releases/latest/download/terragrunt_linux_amd64 -o /usr/local/bin/terragrunt
   chmod +x /usr/local/bin/terragrunt
   ```

3. **libvirt/KVM** (Linux only):
   ```bash
   # Ubuntu/Debian
   sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils

   # Fedora
   sudo dnf install @virtualization

   # Start libvirtd
   sudo systemctl enable --now libvirtd
   ```

4. **Ubuntu Cloud Image**:
   ```bash
   # Download to libvirt images directory
   cd /var/lib/libvirt/images
   sudo wget https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img
   ```

## Usage

1. **Generate VM configuration** using `ucli`:
   ```bash
   ucli
   # Select Create tab, fill in the wizard, click Generate
   ```

2. **Navigate to generated config**:
   ```bash
   cd tf/<vm-name>
   ```

3. **Initialize and apply**:
   ```bash
   terragrunt init
   terragrunt plan
   terragrunt apply
   ```

4. **Connect to VM**:
   ```bash
   # Get outputs
   terragrunt output

   # SSH (once IP is available)
   ssh ubuntu@<ip>

   # Console access
   virsh console <vm-name>
   ```

## Managing VMs

```bash
# List running VMs
virsh list

# Stop a VM
virsh shutdown <vm-name>

# Start a VM
virsh start <vm-name>

# Delete a VM (from its tf directory)
cd tf/<vm-name>
terragrunt destroy
```

## Customization

The generated `terragrunt.hcl` in each VM directory can be customized:

```hcl
inputs = {
  vm_name         = "my-server"
  vcpu_count      = 4          # Increase CPUs
  memory_mb       = 8192       # Increase memory
  disk_size_gb    = 100        # Increase disk
  autostart       = true       # Start on host boot
  # ... other options
}
```

Re-run `terragrunt apply` after changes.
