# Ubuntu Desktop Setup Guide

This guide covers setting up an Ubuntu desktop workstation using this project, whether you're doing a fresh install with a custom ISO or configuring an existing Ubuntu installation.

## Option 1: Fresh Install with Custom ISO

Use this option when installing Ubuntu on a new machine or completely reinstalling.

### Step 1: Generate Your Configuration

```bash
# Clone the repository on any machine with Go installed
git clone https://github.com/jaspreet-dot-casa/cloud-init.git
cd cloud-init

# Build the CLI
make build-cli

# Run the TUI to generate configuration
./bin/ucli
```

In the TUI:
1. Press `2` to go to the **Create** tab
2. Select **Bootable USB** and press Enter
3. Follow the wizard to configure packages and settings

Alternatively, use the command-line wizard:
```bash
./bin/ucli generate
# Select "Bootable ISO" as output format
```

### Step 2: Build the ISO

```bash
# From the TUI: Press 3 for ISO tab, configure options, press Enter to build

# Or from command line:
./bin/ucli build-iso --source ubuntu-24.04-live-server-amd64.iso
```

The ISO builder will:
- Download the Ubuntu base ISO if not provided
- Inject your cloud-init configuration
- Create a bootable ISO at `output/ubuntu-autoinstall.iso`

### Step 3: Create Bootable USB

```bash
# On Linux
sudo dd if=output/ubuntu-autoinstall.iso of=/dev/sdX bs=4M status=progress

# On macOS
diskutil unmountDisk /dev/diskN
sudo dd if=output/ubuntu-autoinstall.iso of=/dev/rdiskN bs=4m
```

Replace `/dev/sdX` or `/dev/diskN` with your USB drive (check with `lsblk` on Linux or `diskutil list` on macOS).

### Step 4: Install Ubuntu

1. Boot from the USB drive
2. Select "Install Ubuntu Server" (autoinstall will proceed automatically)
3. Wait for installation to complete
4. Remove USB and reboot

Your machine will boot with all configured packages installed and ready to use.

## Option 2: Configure an Existing Ubuntu Installation

Use this option when you already have Ubuntu installed and want to add the development tools.

### Quick Setup

```bash
# Clone the repository
git clone https://github.com/jaspreet-dot-casa/cloud-init.git ~/cloud-init
cd ~/cloud-init

# Run the installer
bash scripts/cloud-init/install-all.sh
```

### Customized Setup

For more control over what gets installed:

```bash
cd ~/cloud-init

# Option A: Use the interactive wizard
make build-cli
./bin/ucli generate
# Select packages interactively

# Option B: Edit config.env manually
nano config.env
# Set INSTALL_<PACKAGE>=true/false for each package

# Apply the configuration
make update
```

### Verify Installation

```bash
# Check all tools are installed
make verify-cloud

# Or check individual tools
which starship && starship --version
which zoxide && zoxide --version
which rg && rg --version
```

## Post-Installation Steps

### 1. Log Out and Back In

Shell changes (zsh, starship, zoxide) require a new session:
```bash
exit
# Log back in
```

### 2. Configure Git (if not done during wizard)

```bash
git config --global user.name "Your Name"
git config --global user.email "your@email.com"
```

### 3. Set Up Tailscale (Optional)

If you enabled Tailscale during setup:
```bash
# Check Tailscale status
tailscale status

# If not authenticated, run:
sudo tailscale up
```

### 4. Docker Access

If Docker was installed, ensure your user can access it:
```bash
# This should work without sudo after logging out/in
docker ps
```

If you get permission denied:
```bash
sudo usermod -aG docker $USER
# Log out and back in
```

## Desktop-Specific Packages

For Ubuntu Desktop (not server), you might want additional tools:

### GUI Applications

These aren't included by default but can be added:

```bash
# VS Code
wget -qO- https://packages.microsoft.com/keys/microsoft.asc | gpg --dearmor > packages.microsoft.gpg
sudo install -o root -g root -m 644 packages.microsoft.gpg /etc/apt/keyrings/
echo "deb [arch=amd64 signed-by=/etc/apt/keyrings/packages.microsoft.gpg] https://packages.microsoft.com/repos/code stable main" | sudo tee /etc/apt/sources.list.d/vscode.list
sudo apt update && sudo apt install code

# Chrome
wget https://dl.google.com/linux/direct/google-chrome-stable_current_amd64.deb
sudo dpkg -i google-chrome-stable_current_amd64.deb
```

### Terminal Emulator Recommendations

The project configures zsh with starship. For the best experience, use a modern terminal:

- **Alacritty** - GPU-accelerated, fast, configurable
- **Kitty** - GPU-accelerated with ligature support
- **Wezterm** - Cross-platform with Lua config
- **GNOME Terminal** - Built into Ubuntu, works well

```bash
# Install Alacritty
sudo add-apt-repository ppa:aslatter/ppa
sudo apt update && sudo apt install alacritty
```

## Keeping Your System Updated

### Update Installed Packages

```bash
cd ~/cloud-init
git pull                    # Get latest configurations
make update                 # Apply updates
```

### Preview Changes Before Applying

```bash
make update-dry
```

### Update Only Specific Packages

```bash
# Run individual package scripts
bash scripts/packages/starship.sh update
bash scripts/packages/lazygit.sh update
```

## Troubleshooting

### Starship prompt not showing

Ensure your terminal supports Unicode and has a Nerd Font installed:
```bash
# Install a Nerd Font
wget https://github.com/ryanoasis/nerd-fonts/releases/download/v3.1.1/FiraCode.zip
unzip FiraCode.zip -d ~/.local/share/fonts
fc-cache -fv
```

Then configure your terminal to use "FiraCode Nerd Font".

### zoxide not working

Make sure you've logged out and back in. Check it's in your PATH:
```bash
which zoxide
echo $PATH | grep -q ".local/bin" && echo "OK" || echo "Add ~/.local/bin to PATH"
```

### Shell not changing to zsh

```bash
# Check current shell
echo $SHELL

# Change default shell
chsh -s $(which zsh)

# Log out and back in
```

### fzf key bindings not working

Source the fzf config:
```bash
# Add to ~/.zshrc if not present
[ -f ~/.fzf.zsh ] && source ~/.fzf.zsh
```
