package cloudinit

import (
	"strings"
	"testing"
)

func TestTemplateNotEmpty(t *testing.T) {
	if Template == "" {
		t.Error("Template should not be empty")
	}
}

func TestTemplateContainsCloudConfig(t *testing.T) {
	if !strings.HasPrefix(Template, "#cloud-config") {
		t.Error("Template should start with #cloud-config")
	}
}

func TestTemplateContainsPlaceholders(t *testing.T) {
	placeholders := []string{
		"${USERNAME}",
		"${HOSTNAME}",
		"${SSH_PUBLIC_KEY}",
		"${USER_NAME}",
		"${USER_EMAIL}",
		"${REPO_URL}",
		"${REPO_BRANCH}",
	}

	for _, placeholder := range placeholders {
		if !strings.Contains(Template, placeholder) {
			t.Errorf("Template should contain placeholder %s", placeholder)
		}
	}
}

func TestTemplateContainsRuncmd(t *testing.T) {
	if !strings.Contains(Template, "runcmd:") {
		t.Error("Template should contain runcmd section")
	}
}
