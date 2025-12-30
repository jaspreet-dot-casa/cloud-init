package tui

// OutputMode represents what the CLI should generate.
type OutputMode string

const (
	OutputConfigOnly  OutputMode = "config"    // Just config.env + secrets.env
	OutputCloudInit   OutputMode = "cloudinit" // + cloud-init.yaml
	OutputBootableISO OutputMode = "bootable"  // + bootable ISO for bare metal
	OutputSeedISO     OutputMode = "seed"      // + seed ISO for libvirt (future)
)

// UserConfig holds user configuration inputs.
type UserConfig struct {
	Username      string
	Hostname      string
	SSHPublicKeys []string // Multiple SSH keys supported
	FullName      string   // Git commit name
	Email         string   // Git commit email
	MachineName   string   // Display name for the machine user (can differ from git name)
}

// OptionalConfig holds optional service configuration.
type OptionalConfig struct {
	GithubUser   string
	TailscaleKey string
	GithubPAT    string
}

// ISOConfig holds ISO generation configuration.
type ISOConfig struct {
	SourcePath    string // Path to source Ubuntu ISO
	UbuntuVersion string // "22.04" or "24.04"
	StorageLayout string // "lvm", "direct", or "zfs"
}

// FormResult holds all collected user input.
type FormResult struct {
	User             UserConfig
	SelectedPackages []string
	Optional         OptionalConfig
	OutputMode       OutputMode
	ISO              ISOConfig // ISO options (if OutputBootableISO)
}

// FormOptions configures the behavior of RunForm.
type FormOptions struct {
	// SkipOutputMode skips the output mode question (used by ucli create
	// where target is selected first).
	SkipOutputMode bool
}
