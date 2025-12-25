# Ubuntu Server Setup Made Easy

> Automate your Ubuntu server configuration with a single command. Get a production-ready development environment with modern tools, secure SSH access, and zero manual setup.

## What is this?

This project gives you a **fully configured Ubuntu server** with everything you need for development work - no manual package installation, no configuration files to copy, no hours of setup. Just run one command and you're ready to code.

Perfect for:
- 💻 Setting up new development servers
- 🚀 Deploying to cloud VMs (AWS, DigitalOcean, Linode, etc.)
- 🏠 Configuring home lab machines
- 🔄 Recreating identical environments across multiple servers
- 📦 Getting a consistent dev setup you can version control

## Why use this?

**Instead of spending hours:**
```bash
sudo apt update
sudo apt install git
# ...50 more packages to install manually
# ...countless config files to edit
# ...fighting with PATH issues and dependencies
```

**Just do this:**
```bash
bash scripts/cloud-init/install-all.sh
```

**You get:**
- ✅ Modern shell (Zsh + Oh-My-Zsh + Starship)
- ✅ Essential dev tools (git, docker, neovim, lazygit, lazydocker)
- ✅ Fast CLI utilities (ripgrep, fd, bat, fzf, zoxide)
- ✅ Secure VPN access via Tailscale with built-in 2FA
- ✅ Everything configured and ready to use
- ✅ Reproducible - same setup every time
- ✅ Version controlled - track changes to your config

## Quick Start

### Option 1: Interactive CLI (Recommended)

Use the `ucli` CLI tool for an interactive, guided setup experience:

```bash
# Clone the repository
git clone https://github.com/your-username/your-repo.git ~/cloud-init
cd ~/cloud-init

# Build the CLI
make build-cli

# Run the interactive configuration wizard
./bin/ucli generate
```

The wizard guides you through:
1. User configuration (username, hostname, SSH key)
2. Package selection (choose which tools to install)
3. Optional integrations (GitHub, Tailscale)
4. Output format (config files, cloud-init.yaml, or bootable ISO)

### Option 2: Manual Installation

**What you need:**
- An Ubuntu server (22.04 or 24.04)
- SSH access to it
- 5 minutes

**Three steps to a configured server:**

1. **SSH into your Ubuntu server**
   ```bash
   ssh your-server
   ```

2. **Clone and run the installer**
   ```bash
   git clone https://github.com/your-username/your-repo.git ~/cloud-init
   cd ~/cloud-init
   bash scripts/cloud-init/install-all.sh
   ```

3. **Log out and back in**
   ```bash
   exit
   ssh your-server  # Shell changes take effect
   ```

That's it! You now have a fully configured development server.

### Option 3: Cloud-Init Automation

If you're deploying VMs with cloud-init (AWS, DigitalOcean, etc.), you can automate the entire setup on first boot:

**Using the CLI (recommended):**
```bash
./bin/ucli generate
# Select "Cloud-Init YAML" as output format
```

**Manual method:**
```bash
cd cloud-init/
cp secrets.env.template secrets.env
# Edit secrets.env with your credentials
./generate.sh
```

Use the generated `cloud-init.yaml` with your cloud provider - your VM will automatically configure itself on first boot!

## What Gets Installed

### Core Development Tools

| Tool | What it does | Why you'll love it |
|------|--------------|-------------------|
| **git** | Version control | Industry standard, with `delta` for beautiful diffs |
| **gh** | GitHub CLI | Create PRs, manage issues from terminal |
| **lazygit** | Git TUI | Visual git interface - no more memorizing commands |
| **docker** | Containers | Build and run containers, includes docker-compose |
| **lazydocker** | Docker TUI | Manage containers visually |
| **neovim** | Text editor | Modern vim with better defaults |

### Modern Shell Experience

| Tool | What it does | Why you'll love it |
|------|--------------|-------------------|
| **zsh** | Shell | More powerful than bash, better completion |
| **oh-my-zsh** | Zsh framework | Plugins, themes, instant productivity boost |
| **starship** | Prompt | Beautiful prompt showing git status, language versions |
| **zellij** | Terminal multiplexer | Split terminals, tabs, session management |
| **tmux** | Terminal multiplexer | Alternative to zellij, tried and true |

### Productivity Boosters

| Tool | What it does | Why you'll love it |
|------|--------------|-------------------|
| **ripgrep (rg)** | Fast grep | Search code 10-100x faster than grep |
| **fd** | Fast find | Find files quickly without complex syntax |
| **bat** | Better cat | Syntax highlighting, git integration |
| **fzf** | Fuzzy finder | Interactive file/command search |
| **zoxide** | Smart cd | Jump to directories by typing partial names |
| **btop** | System monitor | Beautiful resource monitoring |
| **jq** | JSON processor | Parse and manipulate JSON like a pro |

### Secure Remote Access

| Tool | What it does | Why you'll love it |
|------|--------------|-------------------|
| **Tailscale** | VPN mesh network | Zero-config VPN, built-in SSH with 2FA |

## How to Use It

### Managing Packages

**Want to add more tools?** Just edit one file:

```bash
# Edit config.env
nano config.env

# Enable what you want
INSTALL_LAZYGIT=true
INSTALL_BTOP=true
INSTALL_ZELLIJ=true

# Apply changes
make update
```

**Preview before applying:**
```bash
make update-dry  # See what will change
```

### Customizing Your Shell

Create `~/.zshrc.local` for machine-specific customizations:

```bash
# Your personal aliases and functions
alias deploy="git push && ssh production 'cd app && git pull && systemctl restart app'"
alias logs="tail -f /var/log/myapp.log"

# Custom environment variables
export EDITOR=nvim
export PROJECT_DIR=~/code
```

### Managing Git Configuration

Two ways to configure Git:

**Option 1: Edit config.env** (recommended)
```bash
nano config.env

# Set these values
USER_NAME="Your Name"
USER_EMAIL="you@example.com"
GIT_DEFAULT_BRANCH="main"

# Apply
make update
```

**Option 2: Use git commands directly**
```bash
git config --global user.name "Your Name"
git config --global user.email "you@example.com"
```

## Secure Remote Access with Tailscale

This setup includes **Tailscale SSH** for secure remote access:

### What's Tailscale?

Tailscale creates a private network between your devices using WireGuard. It's like a VPN, but:
- ✅ Zero configuration - just authenticate
- ✅ Works behind NATs and firewalls
- ✅ Built-in 2FA via your identity provider (Google, GitHub, etc.)
- ✅ No port forwarding needed

### Quick Tailscale Setup

1. **Install Tailscale on your server** (handled by the installer)

2. **Authenticate during setup** (interactive prompt)

3. **Configure ACLs** for SSH access:
   - Go to https://login.tailscale.com/admin/acls
   - Add this to your ACL policy:

   ```json
   {
     "ssh": [
       {
         "action": "check",
         "src": ["autogroup:member"],
         "dst": ["autogroup:self"],
         "users": ["autogroup:nonroot", "root"],
         "checkPeriod": "12h"
       }
     ]
   }
   ```

4. **SSH from any device on your Tailscale network:**
   ```bash
   ssh username@server-name  # Use Tailscale hostname
   ```

**Benefits:**
- No SSH keys to manage
- 2FA required every 12 hours (configurable)
- Access from anywhere securely
- Centralized access control

See [Tailscale Configuration](#network-services) below for advanced features like exit nodes.

## Common Tasks

### Adding a New Server

```bash
# On your new server
git clone https://github.com/your-username/your-repo.git ~/cloud-init
cd ~/cloud-init
bash scripts/cloud-init/install-all.sh
```

### Updating Packages on Multiple Servers

```bash
# On each server
cd ~/cloud-init
git pull
make update
```

### Testing Changes Before Deployment

```bash
# Test in a local VM first
make test-multipass

# Or preview changes
make update-dry
```

### Verifying Installation

```bash
make verify-cloud  # Checks all tools are installed correctly
```

## Advanced Usage

### For Cloud Deployments

If you're using AWS, DigitalOcean, Linode, or any cloud provider that supports cloud-init:

1. **Generate your cloud-init config:**
   ```bash
   cd cloud-init/
   cp secrets.env.template secrets.env
   nano secrets.env  # Add your credentials
   ./generate.sh
   ```

2. **Copy the contents of `cloud-init.yaml`**

3. **Paste into your cloud provider's "user data" field** when creating a VM

4. **Launch your VM** - it auto-configures on first boot!

### Testing in a VM Before Deployment

Test your configuration safely before applying to production:

```bash
make test-multipass         # Full test with automatic cleanup
make test-multipass-keep    # Keep VM for debugging
multipass shell <vm-name>   # SSH into test VM
```

### Available Make Commands

```bash
make help             # Show all commands
make update           # Update packages (idempotent, safe to re-run)
make update-dry       # Preview changes without applying
make verify-cloud     # Verify installation
make test-multipass   # Test in VM
make test-syntax      # Validate scripts
make shellcheck       # Lint scripts
```

## CLI Tool (ucli)

The `ucli` CLI provides an interactive way to configure and generate cloud-init configurations.

### Installation

```bash
# Build from source
make build-cli

# Or install to your GOPATH
make install-cli

# Verify installation
./bin/ucli --version
```

### Commands

#### `ucli generate` - Interactive Configuration

Launch the interactive TUI wizard to configure your server:

```bash
./bin/ucli generate
```

The wizard walks you through:

| Step | What You Configure |
|------|-------------------|
| **User Setup** | Username, hostname, SSH public key, git name/email |
| **Package Selection** | Choose which tools to install (all enabled by default) |
| **Optional Services** | GitHub username, Tailscale auth key, GitHub PAT |
| **Output Mode** | Config files only, cloud-init.yaml, or bootable ISO |

**Output files generated:**
- `config.env` - Package enables, git settings, Tailscale options
- `cloud-init/secrets.env` - Credentials (SSH keys, auth tokens)

#### `ucli packages` - List Available Packages

See all packages that can be installed:

```bash
./bin/ucli packages
```

Example output:
```
Found 9 packages:

CLI Tools:
  - lazygit: A simple terminal UI for git commands
  - btop: Resource monitor
  - yq: YAML processor

Shell & Terminal:
  - starship: Cross-shell prompt
  - zoxide: Smarter cd command

Docker & Containers:
  - lazydocker: Docker TUI
  - docker: Container runtime
```

#### `ucli validate` - Validate Configuration

Check your config files for errors:

```bash
./bin/ucli validate
```

#### `ucli build` - Non-Interactive Build

Generate cloud-init config from existing files (useful for CI/CD):

```bash
./bin/ucli build
```

### CLI Development

```bash
# Run tests
make test-cli

# Build and run interactively
make run-cli

# Clean build artifacts
make clean-cli
```

## Troubleshooting

### "Docker permission denied"

**Problem:** You can't run docker commands without sudo.

**Solution:** Log out and back in after installation. Docker group changes require a new login.

```bash
exit
ssh your-server  # Log back in
docker ps        # Should work now
```

### "Command not found" for installed tools

**Problem:** Tools were installed but aren't in your PATH.

**Solution:** Log out and back in. Shell changes require a new session.

### Tailscale SSH not working

**Checklist:**
1. Is Tailscale running? `tailscale status`
2. Have you configured ACLs? https://login.tailscale.com/admin/acls
3. Are you on the same Tailscale network? `tailscale ip -4`
4. Try using the Tailscale IP directly: `ssh username@100.x.y.z`

Still stuck? Check the logs:
```bash
sudo journalctl -u tailscaled | grep -i ssh
```

## Network Services

### Tailscale VPN + SSH

This setup uses **Tailscale SSH** exclusively - no traditional OpenSSH server.

#### Configuration

Edit `config/tailscale.conf` to customize:

```bash
# Enable Tailscale SSH (replaces traditional OpenSSH)
TAILSCALE_SSH_ENABLED=true

# Advertise as exit node (route traffic through this server)
TAILSCALE_ADVERTISE_EXIT_NODE=true

# SSH Check Mode (require 2FA via identity provider)
TAILSCALE_SSH_CHECK_MODE=true

# How often to require re-authentication
TAILSCALE_SSH_CHECK_PERIOD="12h"
```

#### SSH Access with 2FA

**Setup ACLs** (one-time):
1. Go to https://login.tailscale.com/admin/acls
2. Add SSH rules:

```json
{
  "ssh": [
    {
      "action": "check",
      "src": ["autogroup:member"],
      "dst": ["autogroup:self"],
      "users": ["autogroup:nonroot", "root"],
      "checkPeriod": "12h"
    }
  ]
}
```

**What this means:**
- Any Tailscale user can SSH to their own devices
- 2FA required via your identity provider (Google, GitHub, etc.)
- Re-authenticate every 12 hours
- Can become any user or root

**Usage:**
```bash
# SSH from another machine on your Tailscale network
ssh username@server-name

# Or use Tailscale IP
ssh username@100.x.y.z
```

#### Exit Node (Route Traffic Through Server)

Turn your server into a VPN exit node:

1. **Enable in Tailscale admin console:**
   - Go to https://login.tailscale.com/admin/machines
   - Click your server
   - Edit route settings → Enable "Use as exit node"

2. **Use from any device:**
   ```bash
   tailscale up --exit-node=server-name
   ```

3. **Verify it's working:**
   ```bash
   curl ifconfig.me  # Should show your server's IP
   ```

**Why use an exit node?**
- Browse with your server's IP address
- Access region-restricted content
- Secure public WiFi traffic

#### Useful Commands

```bash
# Check status
tailscale status

# Get your Tailscale IPs
tailscale ip

# Test connection to another device
tailscale ping other-device

# Enable exit node
tailscale up --exit-node=server-name

# Disable exit node
tailscale up --exit-node=
```

## Project Structure

```
cloud-init/
├── cmd/
│   └── ucli/                       # CLI entry point
│       ├── main.go                 # Main CLI with cobra commands
│       └── main_test.go            # CLI tests
├── pkg/
│   ├── config/                     # Configuration generation
│   │   ├── config.go               # FullConfig type
│   │   ├── writer.go               # Write config.env/secrets.env
│   │   └── writer_test.go
│   ├── packages/                   # Package discovery
│   │   ├── package.go              # Package/Registry types
│   │   ├── discovery.go            # Scan scripts/packages/
│   │   └── discovery_test.go
│   └── tui/                        # Interactive TUI
│       ├── form.go                 # huh form implementation
│       ├── styles.go               # Lipgloss theming
│       └── form_test.go
├── config/
│   └── tailscale.conf              # Tailscale settings
├── cloud-init/
│   ├── cloud-init.template.yaml    # Template for cloud deployments
│   ├── secrets.env.template        # Your credentials (copy and edit)
│   └── generate.sh                 # Generate cloud-init.yaml
├── scripts/
│   ├── cloud-init/
│   │   └── install-all.sh          # Main installer
│   ├── packages/                   # Per-package installers
│   └── shared/                     # Shared config scripts
├── tests/
│   └── multipass/                  # VM testing
├── bin/                            # Built binaries (gitignored)
├── go.mod                          # Go module definition
├── go.sum                          # Go dependencies
├── config.env                      # Package enables, git config
└── Makefile                        # Automation commands
```

## FAQ

**Q: Can I use this on an existing server with stuff already installed?**

A: Yes! All scripts are idempotent (safe to run multiple times). They'll skip what's already installed and only add what's missing.

**Q: What if I don't want some of the packages?**

A: Edit `config.env` and set packages to `false`. For example: `INSTALL_LAZYGIT=false`

**Q: Does this work on Ubuntu 22.04 or just 24.04?**

A: Both work. 24.04 is recommended for the latest features.

**Q: Can I customize the configuration?**

A: Absolutely! All scripts and configs are in version control. Fork the repo and customize to your needs.

**Q: What's the difference between cloud-init and manual installation?**

A: Same result, different timing:
- **cloud-init**: Auto-configures on first VM boot (hands-off)
- **Manual**: You SSH in and run the script yourself (more control)

**Q: Do I need Tailscale?**

A: No, but it's highly recommended for secure remote access. You can disable it in `config.env` if you prefer traditional OpenSSH.

**Q: How do I contribute or report issues?**

A: Open an issue or PR on GitHub! Contributions welcome.

## License

[Add your license here]

## Credits

Built with ❤️ for developers who value automation and reproducibility.
