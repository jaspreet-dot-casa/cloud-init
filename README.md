# ucli - Ubuntu Server Setup CLI

Automated Ubuntu server configuration with modern dev tools. Zero manual setup.

## Quick Install

Download the latest release and run:

```bash
# Auto-detect OS and architecture
curl -fsSL https://github.com/jaspreet-dot-casa/cloud-init/releases/latest/download/ucli-$(uname -s | tr '[:upper:]' '[:lower:]')-$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/') -o ucli
chmod +x ucli
./ucli
```

Or download manually from [Releases](https://github.com/jaspreet-dot-casa/cloud-init/releases).

## What You Get

- **Modern shell**: zsh + Oh-My-Zsh + Starship prompt
- **Dev tools**: git, docker, neovim, lazygit, lazydocker
- **Fast CLI**: ripgrep, fd, bat, fzf, zoxide
- **Secure VPN**: Tailscale with SSH support

## Usage

Run `./ucli` to launch the interactive TUI:

```
┌─────────────────────────────────────────────────────────────────┐
│  ucli - Cloud-Init VM Manager                          [q]uit  │
├─────────┬───────────┬───────────┬──────────┬────────────────────┤
│ [1] VMs │ [2] Create│ [3] ISO   │ [4] Doctor│ [5] Settings      │
├─────────┴───────────┴───────────┴──────────┴────────────────────┤
```

**Tabs:**
- **VMs** - Manage Terraform/libvirt VMs
- **Create** - Launch new VMs or generate cloud-init config
- **ISO** - Build bootable autoinstall ISOs
- **Doctor** - Check and install dependencies
- **Settings** - Configure CLI preferences

### Commands

```bash
./ucli              # Launch TUI
./ucli init .       # Initialize with current directory as project
./ucli packages     # List available packages
./ucli --version    # Show version
```

## Managing VMs with Terraform

ucli manages VMs using Terraform with the dmacvicar/libvirt provider. Each VM gets its own isolated Terraform state in the `tf/<vm-name>/` directory.

### Prerequisites

- Linux host with KVM/libvirt installed
- Terraform installed
- Ubuntu cloud image downloaded

```bash
# Install libvirt (Ubuntu)
sudo apt install qemu-kvm libvirt-daemon-system libvirt-clients

# Download Ubuntu cloud image
wget https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img \
  -O /var/lib/libvirt/images/noble-server-cloudimg-amd64.img
```

### Initialize Project

Before using the TUI, initialize ucli with your project path:

```bash
# From the cloud-init repository directory
./ucli init .

# Or specify an absolute path
./ucli init /path/to/cloud-init
```

This creates `~/.config/ucli/config.yaml` with your project path. You can now run `ucli` from anywhere.

### Create a VM

1. Launch the TUI: `./ucli`
2. Press `2` to go to the **Create** tab
3. Select **Terraform/libvirt** as the target
4. Configure VM settings (name, CPU, memory, disk)
5. Configure SSH keys and packages
6. Review and confirm deployment

The deployer will:
- Create `tf/<vm-name>/` directory
- Copy Terraform templates
- Generate `cloud-init.yaml` and `terraform.tfvars`
- Run `terraform init`, `plan`, and `apply`

### VM Lifecycle

From the **VMs** tab (press `1`):

| Key | Action |
|-----|--------|
| `s` | Start VM (sets `running=true`, runs `terraform apply`) |
| `S` | Stop VM (sets `running=false`, runs `terraform apply`) |
| `d` | Delete VM (runs `terraform destroy`) |
| `c` | Show console command |
| `x` | Show SSH command |
| `r` | Refresh VM list |

All operations use Terraform for consistent state management.

### Directory Structure

```
tf/
├── .gitignore              # Ignores state files
├── web-server/             # Each VM has its own directory
│   ├── main.tf             # Copied from terraform/
│   ├── variables.tf
│   ├── outputs.tf
│   ├── terraform.tfvars    # Generated settings
│   ├── cloud-init.yaml     # Generated cloud-init
│   ├── .terraform/         # Terraform providers (gitignored)
│   └── terraform.tfstate   # Terraform state (gitignored)
└── dev-box/
    └── ...
```

### Manual Terraform Commands

You can also manage VMs directly with Terraform:

```bash
cd tf/my-vm

# View current state
terraform show

# Start/stop by editing terraform.tfvars
# Change: running = true  →  running = false
terraform apply

# Destroy VM
terraform destroy

# Refresh state
terraform apply -refresh-only
```

### Configuration

Global config is stored at `~/.config/ucli/config.yaml`:

```yaml
project_path: /path/to/cloud-init
images_dir: ~/Downloads
cloud_images:
  - id: noble-amd64
    name: Ubuntu 24.04 LTS
    path: /var/lib/libvirt/images/noble-server-cloudimg-amd64.img
preferences:
  auto_verify: true
```

## Applying to Existing Ubuntu

Generate config and apply to an already-running Ubuntu desktop/server:

```bash
# Generate cloud-init.yaml via TUI (Create tab → Config Only)
./ucli

# Then apply on your Ubuntu machine:
sudo mkdir -p /var/lib/cloud/seed/nocloud
sudo cp cloud-init.yaml /var/lib/cloud/seed/nocloud/user-data
echo 'instance-id: manual' | sudo tee /var/lib/cloud/seed/nocloud/meta-data
sudo cloud-init clean --logs
sudo cloud-init init --local && sudo cloud-init init
sudo cloud-init modules --mode=config
sudo cloud-init modules --mode=final
```

Or simply run the install scripts directly:

```bash
bash scripts/cloud-init/install-all.sh
```

## Build from Source

```bash
git clone https://github.com/jaspreet-dot-casa/cloud-init.git
cd cloud-init
make build-cli
./bin/ucli
```

## Documentation

- [Terraform VM Guide](docs/terraform-vms.md) - Managing VMs with libvirt
- [Desktop Setup](docs/desktop-setup.md) - Ubuntu desktop configuration

## License

MIT
