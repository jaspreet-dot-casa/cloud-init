package doctor

import "runtime"

// groupDefinitions defines the check groups with their metadata.
var groupDefinitions = map[string]struct {
	Name        string
	Description string
	Platform    string
	CheckIDs    []string
}{
	GroupTerraform: {
		Name:        "Terraform/libvirt",
		Description: "Required for creating VMs via Terraform with libvirt provider",
		Platform:    PlatformLinux, // libvirt is Linux-only
		CheckIDs:    []string{IDTerraform, IDLibvirt, IDVirsh, IDQemuKVM, IDCloudImage},
	},
	GroupMultipass: {
		Name:        "Multipass",
		Description: "Required for creating local Ubuntu VMs via Multipass",
		Platform:    "", // Works on both platforms
		CheckIDs:    []string{IDMultipass},
	},
	GroupISO: {
		Name:        "ISO/USB",
		Description: "Required for creating bootable ISOs",
		Platform:    "", // Works on both platforms
		CheckIDs:    []string{IDXorriso},
	},
}

// GetGroups returns all check groups applicable to the current platform.
func GetGroups() []CheckGroup {
	platform := runtime.GOOS
	var groups []CheckGroup

	for groupID, def := range groupDefinitions {
		// Skip if group is for a different platform
		if def.Platform != "" && def.Platform != platform {
			continue
		}

		group := CheckGroup{
			ID:          groupID,
			Name:        def.Name,
			Description: def.Description,
			Platform:    def.Platform,
		}
		groups = append(groups, group)
	}

	return groups
}

// GetGroupDefinition returns the definition for a specific group.
func GetGroupDefinition(groupID string) (struct {
	Name        string
	Description string
	Platform    string
	CheckIDs    []string
}, bool) {
	def, ok := groupDefinitions[groupID]
	return def, ok
}

// GetAllGroupIDs returns all group IDs.
func GetAllGroupIDs() []string {
	return []string{GroupTerraform, GroupMultipass, GroupISO}
}
