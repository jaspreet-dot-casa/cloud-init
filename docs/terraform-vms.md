# Terraform/libvirt VM Management

This guide covers creating and managing VMs using Terraform with the libvirt provider, integrated with the ucli TUI.

## Overview

The project uses Terraform to create VMs on KVM/libvirt with cloud-init for automatic provisioning. The TUI provides:

- **VM List view** - See all VMs, their status, IP addresses
- **Create view** - Launch VMs with guided configuration
- **Actions** - Start, stop, delete VMs directly from the TUI

## Prerequisites

### 1. Install KVM/libvirt

```bash
# Ubuntu/Debian
sudo apt update
sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils virtinst

# Add user to libvirt group
sudo usermod -aG libvirt $USER

# Log out and back in, then verify
virsh list --all
```

### 2. Install Terraform

```bash
# Using tfenv (recommended)
git clone https://github.com/tfutils/tfenv.git ~/.tfenv
echo 'export PATH="$HOME/.tfenv/bin:$PATH"' >> ~/.bashrc
source ~/.bashrc
tfenv install 1.6.0
tfenv use 1.6.0

# Verify
terraform --version
```

### 3. Download Ubuntu Cloud Image

```bash
# Ubuntu 24.04 (Noble Numbat)
wget https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img
sudo mv noble-server-cloudimg-amd64.img /var/lib/libvirt/images/

# Or Ubuntu 22.04 (Jammy Jellyfish)
wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img
sudo mv jammy-server-cloudimg-amd64.img /var/lib/libvirt/images/
```

### 4. Configure libvirt Networking

Ensure the default network is running:
```bash
# Check network status
virsh net-list --all

# If not active, start it
virsh net-start default
virsh net-autostart default
```

## Using the TUI

### Launch the TUI

```bash
cd ~/cloud-init
make build-cli
./bin/ucli
```

### Tab 1: VM List

View all VMs managed by Terraform:

```
┌─────────────────────────────────────────────────────────────────┐
│  VMs (Terraform/libvirt)                    Auto-refresh: 5s   │
├─────────────────────────────────────────────────────────────────┤
│  NAME              STATUS     IP              CPU  MEM   DISK   │
│  ────────────────  ─────────  ──────────────  ───  ────  ────   │
│▸ dev-server       running    192.168.122.10   2   4GB   20GB   │
│  test-vm          stopped    -                2   2GB   10GB   │
│  staging          running    192.168.122.15   4   8GB   40GB   │
├─────────────────────────────────────────────────────────────────┤
│ [s]tart  [S]top  [d]elete  [c]onsole  [Enter] details  [?] help │
└─────────────────────────────────────────────────────────────────┘
```

**Key bindings:**
- `↑/↓` or `j/k` - Navigate VMs
- `s` - Start stopped VM
- `S` - Stop running VM
- `d` - Delete VM (with confirmation)
- `c` - Open virsh console
- `Enter` - View VM details
- `r` - Refresh list

### Tab 2: Create VM

Create a new VM with Terraform:

1. Press `2` to switch to Create tab
2. Select **Terraform/libvirt** and press Enter
3. Complete the wizard:
   - **VM Name** - Unique name for the VM
   - **CPUs** - 1, 2, or 4
   - **Memory** - 2GB, 4GB, or 8GB
   - **Disk** - 10GB, 20GB, or 40GB
   - **Ubuntu Image** - Path to cloud image
   - **Packages** - Select what to install

4. Review and confirm the Terraform plan
5. Wait for VM creation

After creation, the TUI returns and the VM appears in the list.

## Command-Line Usage

For scripting or non-interactive use:

### Create VM

```bash
# Interactive wizard
./bin/ucli create

# Or use Terraform directly
cd terraform
terraform init
terraform plan -out=tfplan
terraform apply tfplan
```

### Get VM Info

```bash
# From terraform
cd terraform
terraform output vm_ip
terraform output ssh_command

# From virsh
virsh list --all
virsh dominfo <vm-name>
```

### Connect to VM

```bash
# SSH (after cloud-init completes)
ssh ubuntu@$(terraform -chdir=terraform output -raw vm_ip)

# Console access
virsh console <vm-name>
```

### Stop/Start VM

```bash
# Using virsh (faster)
virsh shutdown <vm-name>
virsh start <vm-name>

# Force stop
virsh destroy <vm-name>
```

### Delete VM

```bash
# Using terraform (removes state too)
cd terraform
terraform destroy -target=libvirt_domain.<vm-name>

# Using virsh only
virsh destroy <vm-name>
virsh undefine <vm-name> --remove-all-storage
```

## Terraform Configuration

### Variables

The VM can be customized via `terraform/terraform.tfvars`:

```hcl
# VM Configuration
vm_name      = "dev-server"
memory_mb    = 4096
vcpu_count   = 2
disk_size_gb = 20

# Ubuntu Image
ubuntu_image_path = "/var/lib/libvirt/images/noble-server-cloudimg-amd64.img"

# Cloud-init
cloud_init_file = "../cloud-init/cloud-init.yaml"

# libvirt settings
libvirt_uri  = "qemu:///system"
storage_pool = "default"
network_name = "default"
autostart    = false
```

### Multiple VMs

To create multiple VMs, use Terraform workspaces or separate state:

```bash
# Using workspaces
cd terraform
terraform workspace new dev
terraform apply -var="vm_name=dev-server"

terraform workspace new staging
terraform apply -var="vm_name=staging-server"

# List workspaces
terraform workspace list
```

### Remote libvirt

Connect to remote libvirt over SSH:

```bash
# In terraform.tfvars
libvirt_uri = "qemu+ssh://user@remote-host/system"
```

Ensure SSH key authentication is set up to the remote host.

## Cloud-Init Integration

VMs are provisioned using cloud-init. The configuration flow:

1. Generate cloud-init config:
   ```bash
   ./bin/ucli generate
   ```

2. Terraform uses `cloud-init/cloud-init.yaml`

3. On first boot, cloud-init:
   - Creates users with SSH keys
   - Installs packages
   - Configures shell and tools
   - Sets up Tailscale (if enabled)

### Checking Cloud-Init Status

Inside the VM:
```bash
# Check status
cloud-init status

# View logs
sudo cat /var/log/cloud-init-output.log

# Wait for completion
cloud-init status --wait
```

## Troubleshooting

### Permission denied on libvirt socket

```bash
# Add user to libvirt group
sudo usermod -aG libvirt $USER
# Log out and back in
```

### VM not getting IP address

1. Check the network is running:
   ```bash
   virsh net-list
   virsh net-start default
   ```

2. Check DHCP leases:
   ```bash
   virsh net-dhcp-leases default
   ```

3. Check cloud-init inside VM:
   ```bash
   virsh console <vm-name>
   # Login and check network
   ip addr
   ```

### Terraform state out of sync

If VMs exist but aren't in Terraform state:

```bash
cd terraform
terraform import libvirt_domain.vm <vm-name>
terraform refresh
```

### Cloud-init not completing

1. Check cloud-init logs:
   ```bash
   virsh console <vm-name>
   sudo cat /var/log/cloud-init-output.log
   ```

2. Common issues:
   - Network not available during boot
   - Incorrect cloud-init YAML syntax
   - Missing packages in apt sources

### Storage pool errors

```bash
# Check pool status
virsh pool-list --all

# Create default pool if missing
sudo mkdir -p /var/lib/libvirt/images
virsh pool-define-as default dir --target /var/lib/libvirt/images
virsh pool-build default
virsh pool-start default
virsh pool-autostart default
```

## Tips

### Speed Up VM Creation

1. Pre-download the Ubuntu cloud image
2. Use a local storage pool on SSD
3. Reduce cloud-init packages if not all are needed

### Resource Monitoring

```bash
# CPU and memory usage
virt-top

# Storage usage
virsh pool-info default
virsh vol-list default

# Network info
virsh net-info default
```

### Snapshots

```bash
# Create snapshot
virsh snapshot-create-as <vm-name> <snapshot-name>

# List snapshots
virsh snapshot-list <vm-name>

# Restore snapshot
virsh snapshot-revert <vm-name> <snapshot-name>

# Delete snapshot
virsh snapshot-delete <vm-name> <snapshot-name>
```
