// Package images provides cloud image management functionality.
package images

import (
	"fmt"
	"runtime"
)

// Registry provides access to curated cloud images and ISOs.
type Registry struct {
	images map[string]ImageMetadata
}

// NewRegistry creates a new image registry with all known images.
func NewRegistry() *Registry {
	return &Registry{
		images: buildImageRegistry(),
	}
}

// buildImageRegistry constructs the complete curated image catalog.
func buildImageRegistry() map[string]ImageMetadata {
	images := make(map[string]ImageMetadata)

	// Ubuntu Cloud Images & Desktop ISOs
	addUbuntuImages(images)

	// Debian Cloud Images & Desktop ISOs
	addDebianImages(images)

	return images
}

// addUbuntuImages adds all Ubuntu cloud images and desktop ISOs.
func addUbuntuImages(images map[string]ImageMetadata) {
	// Ubuntu 24.04 LTS (Noble Numbat) - Cloud Images
	addImage(images, ImageMetadata{
		ID:          "ubuntu-24.04-amd64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "amd64",
		URL:         "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/noble/current/SHA256SUMS",
		Filename:    "noble-server-cloudimg-amd64.img",
		Description: "Ubuntu 24.04 LTS Server Cloud Image (amd64)",
		Size:        "~700MB",
		LTS:         true,
	})

	addImage(images, ImageMetadata{
		ID:          "ubuntu-24.04-arm64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "arm64",
		URL:         "https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-arm64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/noble/current/SHA256SUMS",
		Filename:    "noble-server-cloudimg-arm64.img",
		Description: "Ubuntu 24.04 LTS Server Cloud Image (arm64)",
		Size:        "~650MB",
		LTS:         true,
	})

	addImage(images, ImageMetadata{
		ID:          "ubuntu-24.04-amd64-minimal",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantMinimal,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "amd64",
		URL:         "https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-amd64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/minimal/releases/noble/release/SHA256SUMS",
		Filename:    "ubuntu-24.04-minimal-cloudimg-amd64.img",
		Description: "Ubuntu 24.04 LTS Minimal Cloud Image (amd64)",
		Size:        "~400MB",
		LTS:         true,
	})

	addImage(images, ImageMetadata{
		ID:          "ubuntu-24.04-arm64-minimal",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantMinimal,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "arm64",
		URL:         "https://cloud-images.ubuntu.com/minimal/releases/noble/release/ubuntu-24.04-minimal-cloudimg-arm64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/minimal/releases/noble/release/SHA256SUMS",
		Filename:    "ubuntu-24.04-minimal-cloudimg-arm64.img",
		Description: "Ubuntu 24.04 LTS Minimal Cloud Image (arm64)",
		Size:        "~350MB",
		LTS:         true,
	})

	// Ubuntu 24.04 LTS - Desktop ISO
	addImage(images, ImageMetadata{
		ID:          "ubuntu-24.04-amd64-desktop",
		Source:      SourceUbuntu,
		Type:        TypeDesktop,
		Variant:     VariantDesktop,
		OS:          "Ubuntu",
		Version:     "24.04",
		Codename:    "noble",
		Arch:        "amd64",
		URL:         "https://releases.ubuntu.com/24.04/ubuntu-24.04-desktop-amd64.iso",
		ChecksumURL: "https://releases.ubuntu.com/24.04/SHA256SUMS",
		Filename:    "ubuntu-24.04-desktop-amd64.iso",
		Description: "Ubuntu 24.04 LTS Desktop (amd64)",
		Size:        "~5.7GB",
		LTS:         true,
	})

	// Ubuntu 23.10 (Mantic Minotaur) - Cloud Images
	addImage(images, ImageMetadata{
		ID:          "ubuntu-23.10-amd64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "23.10",
		Codename:    "mantic",
		Arch:        "amd64",
		URL:         "https://cloud-images.ubuntu.com/mantic/current/mantic-server-cloudimg-amd64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/mantic/current/SHA256SUMS",
		Filename:    "mantic-server-cloudimg-amd64.img",
		Description: "Ubuntu 23.10 Server Cloud Image (amd64)",
		Size:        "~680MB",
		LTS:         false,
	})

	addImage(images, ImageMetadata{
		ID:          "ubuntu-23.10-arm64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "23.10",
		Codename:    "mantic",
		Arch:        "arm64",
		URL:         "https://cloud-images.ubuntu.com/mantic/current/mantic-server-cloudimg-arm64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/mantic/current/SHA256SUMS",
		Filename:    "mantic-server-cloudimg-arm64.img",
		Description: "Ubuntu 23.10 Server Cloud Image (arm64)",
		Size:        "~640MB",
		LTS:         false,
	})

	// Ubuntu 23.10 - Desktop ISO
	addImage(images, ImageMetadata{
		ID:          "ubuntu-23.10-amd64-desktop",
		Source:      SourceUbuntu,
		Type:        TypeDesktop,
		Variant:     VariantDesktop,
		OS:          "Ubuntu",
		Version:     "23.10",
		Codename:    "mantic",
		Arch:        "amd64",
		URL:         "https://releases.ubuntu.com/23.10/ubuntu-23.10-desktop-amd64.iso",
		ChecksumURL: "https://releases.ubuntu.com/23.10/SHA256SUMS",
		Filename:    "ubuntu-23.10-desktop-amd64.iso",
		Description: "Ubuntu 23.10 Desktop (amd64)",
		Size:        "~5.5GB",
		LTS:         false,
	})

	// Ubuntu 22.04 LTS (Jammy Jellyfish) - for backward compatibility
	addImage(images, ImageMetadata{
		ID:          "ubuntu-22.04-amd64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "22.04",
		Codename:    "jammy",
		Arch:        "amd64",
		URL:         "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-amd64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/jammy/current/SHA256SUMS",
		Filename:    "jammy-server-cloudimg-amd64.img",
		Description: "Ubuntu 22.04 LTS Server Cloud Image (amd64)",
		Size:        "~650MB",
		LTS:         true,
	})

	addImage(images, ImageMetadata{
		ID:          "ubuntu-22.04-arm64-server",
		Source:      SourceUbuntu,
		Type:        TypeCloudInit,
		Variant:     VariantServer,
		OS:          "Ubuntu",
		Version:     "22.04",
		Codename:    "jammy",
		Arch:        "arm64",
		URL:         "https://cloud-images.ubuntu.com/jammy/current/jammy-server-cloudimg-arm64.img",
		ChecksumURL: "https://cloud-images.ubuntu.com/jammy/current/SHA256SUMS",
		Filename:    "jammy-server-cloudimg-arm64.img",
		Description: "Ubuntu 22.04 LTS Server Cloud Image (arm64)",
		Size:        "~620MB",
		LTS:         true,
	})
}

// addDebianImages adds all Debian cloud images and desktop ISOs.
func addDebianImages(images map[string]ImageMetadata) {
	// Debian 12 (Bookworm) - Generic Cloud Image
	addImage(images, ImageMetadata{
		ID:          "debian-12-amd64-generic",
		Source:      SourceDebian,
		Type:        TypeCloudInit,
		Variant:     VariantGeneric,
		OS:          "Debian",
		Version:     "12",
		Codename:    "bookworm",
		Arch:        "amd64",
		URL:         "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-amd64.qcow2",
		ChecksumURL: "https://cloud.debian.org/images/cloud/bookworm/latest/SHA512SUMS",
		Filename:    "debian-12-generic-amd64.qcow2",
		Description: "Debian 12 Bookworm Generic Cloud Image (amd64)",
		Size:        "~500MB",
		LTS:         false,
	})

	addImage(images, ImageMetadata{
		ID:          "debian-12-arm64-generic",
		Source:      SourceDebian,
		Type:        TypeCloudInit,
		Variant:     VariantGeneric,
		OS:          "Debian",
		Version:     "12",
		Codename:    "bookworm",
		Arch:        "arm64",
		URL:         "https://cloud.debian.org/images/cloud/bookworm/latest/debian-12-generic-arm64.qcow2",
		ChecksumURL: "https://cloud.debian.org/images/cloud/bookworm/latest/SHA512SUMS",
		Filename:    "debian-12-generic-arm64.qcow2",
		Description: "Debian 12 Bookworm Generic Cloud Image (arm64)",
		Size:        "~480MB",
		LTS:         false,
	})

	// Debian 12 - Desktop ISO (netinst)
	addImage(images, ImageMetadata{
		ID:          "debian-12-amd64-desktop",
		Source:      SourceDebian,
		Type:        TypeDesktop,
		Variant:     VariantDesktop,
		OS:          "Debian",
		Version:     "12",
		Codename:    "bookworm",
		Arch:        "amd64",
		URL:         "https://cdimage.debian.org/debian-cd/current/amd64/iso-dvd/debian-12.8.0-amd64-DVD-1.iso",
		ChecksumURL: "https://cdimage.debian.org/debian-cd/current/amd64/iso-dvd/SHA256SUMS",
		Filename:    "debian-12.8.0-amd64-DVD-1.iso",
		Description: "Debian 12 Bookworm Desktop DVD (amd64)",
		Size:        "~3.7GB",
		LTS:         false,
	})

	// Debian 13 (Trixie) - Generic Cloud Image
	addImage(images, ImageMetadata{
		ID:          "debian-13-amd64-generic",
		Source:      SourceDebian,
		Type:        TypeCloudInit,
		Variant:     VariantGeneric,
		OS:          "Debian",
		Version:     "13",
		Codename:    "trixie",
		Arch:        "amd64",
		URL:         "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-amd64.qcow2",
		ChecksumURL: "https://cloud.debian.org/images/cloud/trixie/latest/SHA512SUMS",
		Filename:    "debian-13-generic-amd64.qcow2",
		Description: "Debian 13 Trixie Generic Cloud Image (amd64) [Testing]",
		Size:        "~520MB",
		LTS:         false,
	})

	addImage(images, ImageMetadata{
		ID:          "debian-13-arm64-generic",
		Source:      SourceDebian,
		Type:        TypeCloudInit,
		Variant:     VariantGeneric,
		OS:          "Debian",
		Version:     "13",
		Codename:    "trixie",
		Arch:        "arm64",
		URL:         "https://cloud.debian.org/images/cloud/trixie/latest/debian-13-generic-arm64.qcow2",
		ChecksumURL: "https://cloud.debian.org/images/cloud/trixie/latest/SHA512SUMS",
		Filename:    "debian-13-generic-arm64.qcow2",
		Description: "Debian 13 Trixie Generic Cloud Image (arm64) [Testing]",
		Size:        "~500MB",
		LTS:         false,
	})

	// Debian 13 - Desktop ISO
	addImage(images, ImageMetadata{
		ID:          "debian-13-amd64-desktop",
		Source:      SourceDebian,
		Type:        TypeDesktop,
		Variant:     VariantDesktop,
		OS:          "Debian",
		Version:     "13",
		Codename:    "trixie",
		Arch:        "amd64",
		URL:         "https://cdimage.debian.org/cdimage/weekly-builds/amd64/iso-dvd/debian-testing-amd64-DVD-1.iso",
		ChecksumURL: "https://cdimage.debian.org/cdimage/weekly-builds/amd64/iso-dvd/SHA256SUMS",
		Filename:    "debian-testing-amd64-DVD-1.iso",
		Description: "Debian 13 Trixie Desktop DVD (amd64) [Testing]",
		Size:        "~3.8GB",
		LTS:         false,
	})
}

// addImage is a helper to add an image to the registry map.
func addImage(images map[string]ImageMetadata, img ImageMetadata) {
	images[img.ID] = img
}

// GetAll returns all registered images.
func (r *Registry) GetAll() []ImageMetadata {
	images := make([]ImageMetadata, 0, len(r.images))
	for _, img := range r.images {
		images = append(images, img)
	}
	return images
}

// GetByID retrieves an image by its ID.
func (r *Registry) GetByID(id string) *ImageMetadata {
	if img, ok := r.images[id]; ok {
		return &img
	}
	return nil
}

// GetBySource returns images filtered by source (Ubuntu or Debian).
func (r *Registry) GetBySource(source ImageSource) []ImageMetadata {
	var result []ImageMetadata
	for _, img := range r.images {
		if img.Source == source {
			result = append(result, img)
		}
	}
	return result
}

// GetByType returns images filtered by type (CloudInit, NoCloud, Desktop).
func (r *Registry) GetByType(imgType ImageType) []ImageMetadata {
	var result []ImageMetadata
	for _, img := range r.images {
		if img.Type == imgType {
			result = append(result, img)
		}
	}
	return result
}

// GroupByOS returns images organized by OS name.
func (r *Registry) GroupByOS() map[string][]ImageMetadata {
	grouped := make(map[string][]ImageMetadata)
	for _, img := range r.images {
		grouped[img.OS] = append(grouped[img.OS], img)
	}
	return grouped
}

// GroupByVersion returns images for a specific OS organized by version.
func (r *Registry) GroupByVersion(os string) map[string][]ImageMetadata {
	grouped := make(map[string][]ImageMetadata)
	for _, img := range r.images {
		if img.OS == os {
			grouped[img.Version] = append(grouped[img.Version], img)
		}
	}
	return grouped
}

// GroupByTypeForOS returns images for an OS grouped by type.
func (r *Registry) GroupByTypeForOS(os string, version string) map[ImageType][]ImageMetadata {
	grouped := make(map[ImageType][]ImageMetadata)
	for _, img := range r.images {
		if img.OS == os && img.Version == version {
			grouped[img.Type] = append(grouped[img.Type], img)
		}
	}
	return grouped
}

// GetOSList returns a list of unique OS names.
func (r *Registry) GetOSList() []string {
	osSet := make(map[string]bool)
	for _, img := range r.images {
		osSet[img.OS] = true
	}

	osList := make([]string, 0, len(osSet))
	for os := range osSet {
		osList = append(osList, os)
	}
	return osList
}

// GetVersionsForOS returns a list of versions for a specific OS.
func (r *Registry) GetVersionsForOS(os string) []string {
	versionSet := make(map[string]bool)
	for _, img := range r.images {
		if img.OS == os {
			versionSet[img.Version] = true
		}
	}

	versions := make([]string, 0, len(versionSet))
	for v := range versionSet {
		versions = append(versions, v)
	}
	return versions
}

// GetTypesForOSVersion returns available image types for an OS version.
func (r *Registry) GetTypesForOSVersion(os string, version string) []ImageType {
	typeSet := make(map[ImageType]bool)
	for _, img := range r.images {
		if img.OS == os && img.Version == version {
			typeSet[img.Type] = true
		}
	}

	types := make([]ImageType, 0, len(typeSet))
	for t := range typeSet {
		types = append(types, t)
	}
	return types
}

// GetImagesForOSVersionType returns images for specific OS, version, and type.
func (r *Registry) GetImagesForOSVersionType(os string, version string, imgType ImageType) []ImageMetadata {
	var result []ImageMetadata
	for _, img := range r.images {
		if img.OS == os && img.Version == version && img.Type == imgType {
			result = append(result, img)
		}
	}
	return result
}

// GetDefaultArch returns the default architecture for the current system.
func GetDefaultArch() string {
	switch runtime.GOARCH {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	default:
		return "amd64" // Default to amd64
	}
}

// Legacy compatibility functions (for existing code that uses old Registry)

// UbuntuRelease represents an Ubuntu release (legacy).
type UbuntuRelease struct {
	Version  string   // e.g., "24.04"
	Codename string   // e.g., "noble"
	Name     string   // e.g., "Ubuntu 24.04 LTS (Noble Numbat)"
	LTS      bool     // Is this an LTS release
	Archs    []string // Supported architectures
}

// CloudImageInfo contains information about a cloud image (legacy).
type CloudImageInfo struct {
	Version  string // Ubuntu version
	Codename string // Ubuntu codename
	Arch     string // Architecture
	URL      string // Download URL
	SHA256   string // SHA256 checksum (may be empty if not known)
	Filename string // Expected filename
}

// KnownReleases contains known Ubuntu releases (legacy).
var KnownReleases = []UbuntuRelease{
	{
		Version:  "24.04",
		Codename: "noble",
		Name:     "Ubuntu 24.04 LTS (Noble Numbat)",
		LTS:      true,
		Archs:    []string{"amd64", "arm64"},
	},
	{
		Version:  "23.10",
		Codename: "mantic",
		Name:     "Ubuntu 23.10 (Mantic Minotaur)",
		LTS:      false,
		Archs:    []string{"amd64", "arm64"},
	},
	{
		Version:  "22.04",
		Codename: "jammy",
		Name:     "Ubuntu 22.04 LTS (Jammy Jellyfish)",
		LTS:      true,
		Archs:    []string{"amd64", "arm64"},
	},
}

// GetReleases returns all known releases (legacy).
func (r *Registry) GetReleases() []UbuntuRelease {
	return KnownReleases
}

// GetLTSReleases returns only LTS releases (legacy).
func (r *Registry) GetLTSReleases() []UbuntuRelease {
	var lts []UbuntuRelease
	for _, rel := range KnownReleases {
		if rel.LTS {
			lts = append(lts, rel)
		}
	}
	return lts
}

// FindRelease finds a release by version or codename (legacy).
func (r *Registry) FindRelease(versionOrCodename string) *UbuntuRelease {
	for i := range KnownReleases {
		rel := &KnownReleases[i]
		if rel.Version == versionOrCodename || rel.Codename == versionOrCodename {
			return rel
		}
	}
	return nil
}

// GetCloudImageInfo returns information about a cloud image (legacy).
func (r *Registry) GetCloudImageInfo(version, arch string) *CloudImageInfo {
	// Try to find in new registry first
	for _, img := range r.images {
		if img.Source == SourceUbuntu && img.Type == TypeCloudInit &&
			img.Variant == VariantServer && img.Version == version && img.Arch == arch {
			return &CloudImageInfo{
				Version:  img.Version,
				Codename: img.Codename,
				Arch:     img.Arch,
				URL:      img.URL,
				Filename: img.Filename,
			}
		}
	}
	return nil
}

// GetAllCloudImages returns all available cloud images (legacy).
func (r *Registry) GetAllCloudImages() []CloudImageInfo {
	var result []CloudImageInfo
	for _, img := range r.images {
		if img.Source == SourceUbuntu && img.Type == TypeCloudInit && img.Variant == VariantServer {
			result = append(result, CloudImageInfo{
				Version:  img.Version,
				Codename: img.Codename,
				Arch:     img.Arch,
				URL:      img.URL,
				Filename: img.Filename,
			})
		}
	}
	return result
}

// GenerateImageID generates a unique ID for a cloud image (legacy).
func GenerateImageID(version, arch string) string {
	return fmt.Sprintf("ubuntu-%s-%s", version, arch)
}
