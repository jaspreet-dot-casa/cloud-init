# Terraform for libvirt/KVM

Provisions Ubuntu VMs with cloud-init configuration using KVM/QEMU via libvirt.

## Prerequisites

1. **libvirt/KVM installed**
   ```bash
   # Ubuntu/Debian
   sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients bridge-utils

   # Fedora
   sudo dnf install qemu-kvm libvirt virt-install
   ```

2. **Terraform installed**
   ```bash
   # Using tfenv (recommended)
   tfenv install 1.6.0
   tfenv use 1.6.0

   # Or direct download from terraform.io
   ```

3. **Ubuntu cloud image**
   ```bash
   # Download Ubuntu 22.04 cloud image
   wget https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img
   sudo mv jammy-server-cloudimg-amd64.img /var/lib/libvirt/images/
   ```

4. **Cloud-init configuration**
   ```bash
   # Generate cloud-init.yaml
   cd ../cloud-init
   cp secrets.env.template secrets.env
   # Edit secrets.env with your values
   ./generate.sh
   ```

## Usage

1. **Initialize Terraform**
   ```bash
   terraform init
   ```

2. **Configure variables** (optional)
   ```bash
   cp terraform.tfvars.example terraform.tfvars
   # Edit terraform.tfvars
   ```

3. **Plan and apply**
   ```bash
   terraform plan
   terraform apply
   ```

4. **Connect to VM**
   ```bash
   # Get IP address
   terraform output vm_ip

   # SSH (after cloud-init completes)
   ssh <username>@<ip-address>

   # Console access
   virsh console ubuntu-server
   ```

## Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `libvirt_uri` | `qemu:///system` | Libvirt connection URI |
| `storage_pool` | `default` | Storage pool name |
| `network_name` | `default` | Network name |
| `vm_name` | `ubuntu-server` | VM name |
| `memory_mb` | `2048` | Memory in MB |
| `vcpu_count` | `2` | Number of vCPUs |
| `disk_size_gb` | `20` | Disk size in GB |
| `autostart` | `false` | Start VM automatically on host boot |
| `ubuntu_image_path` | `/var/lib/libvirt/images/jammy-server-cloudimg-amd64.img` | Ubuntu image path |
| `cloud_init_file` | `../cloud-init/cloud-init.yaml` | Cloud-init file |

## Outputs

After `terraform apply`:

```bash
# VM IP address
terraform output vm_ip

# SSH command
terraform output ssh_command

# Console command
terraform output console_command
```

## Cleanup

```bash
terraform destroy
```

## Troubleshooting

### Permission denied on libvirt
```bash
sudo usermod -aG libvirt $USER
# Log out and back in
```

### Network not found
```bash
# List networks
virsh net-list --all

# Start default network
virsh net-start default
virsh net-autostart default
```

### Storage pool not found
```bash
# List pools
virsh pool-list --all

# Create default pool
virsh pool-define-as default dir --target /var/lib/libvirt/images
virsh pool-start default
virsh pool-autostart default
```

### VM not getting IP
```bash
# Check DHCP leases
virsh net-dhcp-leases default

# Console access to debug
virsh console ubuntu-server
```
