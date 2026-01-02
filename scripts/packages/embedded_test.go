package packages

import (
	"io/fs"
	"strings"
	"testing"
)

func TestScriptsNotEmpty(t *testing.T) {
	entries, err := fs.ReadDir(Scripts, ".")
	if err != nil {
		t.Fatalf("Failed to read embedded scripts: %v", err)
	}

	if len(entries) == 0 {
		t.Error("Scripts should contain at least one file")
	}
}

func TestScriptsContainsExpectedPackages(t *testing.T) {
	expectedPackages := []string{
		"apt.sh",
		"docker.sh",
		"lazygit.sh",
		"starship.sh",
		"zoxide.sh",
	}

	entries, err := fs.ReadDir(Scripts, ".")
	if err != nil {
		t.Fatalf("Failed to read embedded scripts: %v", err)
	}

	fileNames := make(map[string]bool)
	for _, entry := range entries {
		fileNames[entry.Name()] = true
	}

	for _, expected := range expectedPackages {
		if !fileNames[expected] {
			t.Errorf("Expected package script %s not found in embedded scripts", expected)
		}
	}
}

func TestScriptsExcludesTemplate(t *testing.T) {
	entries, err := fs.ReadDir(Scripts, ".")
	if err != nil {
		t.Fatalf("Failed to read embedded scripts: %v", err)
	}

	for _, entry := range entries {
		// _template.sh should be embedded but will be skipped during discovery
		// This test just verifies the embed works, not the filtering
		if !strings.HasSuffix(entry.Name(), ".sh") {
			t.Errorf("Non-.sh file found in embedded scripts: %s", entry.Name())
		}
	}
}

func TestScriptsAreReadable(t *testing.T) {
	entries, err := fs.ReadDir(Scripts, ".")
	if err != nil {
		t.Fatalf("Failed to read embedded scripts: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		content, err := fs.ReadFile(Scripts, entry.Name())
		if err != nil {
			t.Errorf("Failed to read script %s: %v", entry.Name(), err)
			continue
		}

		if len(content) == 0 {
			t.Errorf("Script %s is empty", entry.Name())
		}

		// Check it starts with shebang
		if !strings.HasPrefix(string(content), "#!/") {
			t.Errorf("Script %s should start with shebang", entry.Name())
		}
	}
}

func TestScriptsContainPackageName(t *testing.T) {
	entries, err := fs.ReadDir(Scripts, ".")
	if err != nil {
		t.Fatalf("Failed to read embedded scripts: %v", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || entry.Name() == "_template.sh" {
			continue
		}

		content, err := fs.ReadFile(Scripts, entry.Name())
		if err != nil {
			t.Errorf("Failed to read script %s: %v", entry.Name(), err)
			continue
		}

		if !strings.Contains(string(content), "PACKAGE_NAME=") {
			t.Errorf("Script %s should contain PACKAGE_NAME definition", entry.Name())
		}
	}
}
