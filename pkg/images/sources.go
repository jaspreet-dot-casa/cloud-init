// Package images provides cloud image management functionality.
package images

// ImageSource represents the operating system distribution.
type ImageSource int

const (
	SourceUbuntu ImageSource = iota
	SourceDebian
)

// String returns the string representation of ImageSource.
func (s ImageSource) String() string {
	switch s {
	case SourceUbuntu:
		return "ubuntu"
	case SourceDebian:
		return "debian"
	default:
		return "unknown"
	}
}

// ImageType categorizes images by cloud-init capability.
type ImageType int

const (
	TypeCloudInit ImageType = iota // Has cloud-init support
	TypeNoCloud                     // No cloud-init (nocloud images)
	TypeDesktop                     // Desktop ISOs
)

// String returns the string representation of ImageType.
func (t ImageType) String() string {
	switch t {
	case TypeCloudInit:
		return "cloud-init"
	case TypeNoCloud:
		return "nocloud"
	case TypeDesktop:
		return "desktop"
	default:
		return "unknown"
	}
}

// DisplayName returns the user-friendly display name for ImageType.
func (t ImageType) DisplayName() string {
	switch t {
	case TypeCloudInit:
		return "Cloud Images (cloud-init enabled)"
	case TypeNoCloud:
		return "Cloud Images (no cloud-init)"
	case TypeDesktop:
		return "Desktop ISOs"
	default:
		return "Unknown"
	}
}

// ImageVariant specifies the specific purpose or configuration of an image.
type ImageVariant string

const (
	VariantServer       ImageVariant = "server"       // Ubuntu server cloud image
	VariantMinimal      ImageVariant = "minimal"      // Ubuntu minimal cloud image
	VariantDesktop      ImageVariant = "desktop"      // Desktop ISO
	VariantGeneric      ImageVariant = "generic"      // Debian generic cloud image
	VariantGenericCloud ImageVariant = "genericcloud" // Debian reduced drivers
	VariantNoCloud      ImageVariant = "nocloud"      // Debian no cloud-init
)

// String returns the string representation of ImageVariant.
func (v ImageVariant) String() string {
	return string(v)
}

// DisplayName returns the user-friendly display name for ImageVariant.
func (v ImageVariant) DisplayName() string {
	switch v {
	case VariantServer:
		return "Server"
	case VariantMinimal:
		return "Minimal"
	case VariantDesktop:
		return "Desktop"
	case VariantGeneric:
		return "Generic"
	case VariantGenericCloud:
		return "Generic Cloud"
	case VariantNoCloud:
		return "No Cloud"
	default:
		return string(v)
	}
}

// ImageMetadata contains complete information about a distributable image.
type ImageMetadata struct {
	ID          string       // Unique identifier (e.g., "ubuntu-24.04-amd64-server")
	Source      ImageSource  // Operating system source
	Type        ImageType    // Cloud-init capability
	Variant     ImageVariant // Image variant/purpose
	OS          string       // Display OS name (e.g., "Ubuntu", "Debian")
	Version     string       // Version number (e.g., "24.04", "12")
	Codename    string       // Release codename (e.g., "noble", "bookworm")
	Arch        string       // Architecture (e.g., "amd64", "arm64")
	URL         string       // Download URL
	ChecksumURL string       // Checksum file URL (SHA256SUMS, SHA512SUMS, etc.)
	Filename    string       // Expected filename after download
	Description string       // Human-readable description
	Size        string       // Approximate size (e.g., "~700MB", "~4GB")
	LTS         bool         // Is this an LTS/stable release
}

// GenerateID creates a unique identifier for an image based on its metadata.
func (m *ImageMetadata) GenerateID() string {
	return m.ID
}

// IsCloudInit returns true if this image has cloud-init support.
func (m *ImageMetadata) IsCloudInit() bool {
	return m.Type == TypeCloudInit
}

// DefaultFilename returns the expected filename for this image.
func (m *ImageMetadata) DefaultFilename() string {
	if m.Filename != "" {
		return m.Filename
	}
	// Fallback to generating from metadata
	return m.ID + "-image"
}
