// Package packages provides embedded package installer scripts.
package packages

import "embed"

// Scripts contains all package installer scripts.
// These are parsed to discover available packages and their metadata.
// The actual scripts are not executed from here - they're cloned from git
// during cloud-init execution on the target VM.
//
//go:embed *.sh
var Scripts embed.FS
