// Package images provides cloud image management functionality.
package images

import (
	"fmt"
	"runtime"
)

// UbuntuRelease represents an Ubuntu release.
type UbuntuRelease struct {
	Version  string   // e.g., "24.04"
	Codename string   // e.g., "noble"
	Name     string   // e.g., "Ubuntu 24.04 LTS (Noble Numbat)"
	LTS      bool     // Is this an LTS release
	Archs    []string // Supported architectures
}

// CloudImageInfo contains information about a cloud image.
type CloudImageInfo struct {
	Version  string // Ubuntu version
	Codename string // Ubuntu codename
	Arch     string // Architecture
	URL      string // Download URL
	SHA256   string // SHA256 checksum (may be empty if not known)
	Filename string // Expected filename
}

// BaseURL is the base URL for Ubuntu cloud images.
const BaseURL = "https://cloud-images.ubuntu.com"

// KnownReleases contains known Ubuntu releases.
var KnownReleases = []UbuntuRelease{
	{
		Version:  "24.04",
		Codename: "noble",
		Name:     "Ubuntu 24.04 LTS (Noble Numbat)",
		LTS:      true,
		Archs:    []string{"amd64", "arm64"},
	},
	{
		Version:  "22.04",
		Codename: "jammy",
		Name:     "Ubuntu 22.04 LTS (Jammy Jellyfish)",
		LTS:      true,
		Archs:    []string{"amd64", "arm64"},
	},
	{
		Version:  "20.04",
		Codename: "focal",
		Name:     "Ubuntu 20.04 LTS (Focal Fossa)",
		LTS:      true,
		Archs:    []string{"amd64", "arm64"},
	},
}

// Registry provides access to known cloud images.
type Registry struct {
	releases []UbuntuRelease
}

// NewRegistry creates a new image registry.
func NewRegistry() *Registry {
	return &Registry{
		releases: KnownReleases,
	}
}

// GetReleases returns all known releases.
func (r *Registry) GetReleases() []UbuntuRelease {
	return r.releases
}

// GetLTSReleases returns only LTS releases.
func (r *Registry) GetLTSReleases() []UbuntuRelease {
	var lts []UbuntuRelease
	for _, rel := range r.releases {
		if rel.LTS {
			lts = append(lts, rel)
		}
	}
	return lts
}

// FindRelease finds a release by version or codename.
func (r *Registry) FindRelease(versionOrCodename string) *UbuntuRelease {
	for i := range r.releases {
		rel := &r.releases[i]
		if rel.Version == versionOrCodename || rel.Codename == versionOrCodename {
			return rel
		}
	}
	return nil
}

// GetCloudImageInfo returns information about a cloud image for a specific release and arch.
func (r *Registry) GetCloudImageInfo(version, arch string) *CloudImageInfo {
	rel := r.FindRelease(version)
	if rel == nil {
		return nil
	}

	// Check if arch is supported
	archSupported := false
	for _, a := range rel.Archs {
		if a == arch {
			archSupported = true
			break
		}
	}
	if !archSupported {
		return nil
	}

	// Generate URL
	// Format: https://cloud-images.ubuntu.com/noble/current/noble-server-cloudimg-amd64.img
	filename := fmt.Sprintf("%s-server-cloudimg-%s.img", rel.Codename, arch)
	url := fmt.Sprintf("%s/%s/current/%s", BaseURL, rel.Codename, filename)

	return &CloudImageInfo{
		Version:  rel.Version,
		Codename: rel.Codename,
		Arch:     arch,
		URL:      url,
		Filename: filename,
	}
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

// GetAllCloudImages returns all available cloud images.
func (r *Registry) GetAllCloudImages() []CloudImageInfo {
	var images []CloudImageInfo
	for _, rel := range r.releases {
		for _, arch := range rel.Archs {
			info := r.GetCloudImageInfo(rel.Version, arch)
			if info != nil {
				images = append(images, *info)
			}
		}
	}
	return images
}

// GenerateImageID generates a unique ID for a cloud image.
func GenerateImageID(version, arch string) string {
	return fmt.Sprintf("ubuntu-%s-%s", version, arch)
}
