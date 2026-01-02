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

### Commands

```bash
./ucli              # Launch TUI
./ucli packages     # List available packages
./ucli --version    # Show version
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
- [Tailscale SSH](docs/tailscale-ssh.md) - Secure remote access

## License

MIT
