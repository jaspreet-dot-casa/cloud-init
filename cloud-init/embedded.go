// Package cloudinit provides embedded cloud-init template files.
package cloudinit

import _ "embed"

// Template contains the cloud-init.yaml template with variable placeholders.
// Variables like ${USERNAME}, ${HOSTNAME}, etc. are substituted at generation time.
//
//go:embed cloud-init.template.yaml
var Template string
