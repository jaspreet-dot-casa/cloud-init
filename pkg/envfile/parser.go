// Package envfile provides utilities for parsing shell-style environment files.
package envfile

import (
	"bufio"
	"os"
	"strings"
)

// Parse parses a shell-style env file and returns key-value pairs.
// It handles:
// - KEY=VALUE format
// - KEY="VALUE" and KEY='VALUE' (quotes are stripped)
// - Comments (lines starting with #)
// - Empty lines (skipped)
// - Values containing = signs (only first = is used as delimiter)
func Parse(path string) (map[string]string, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	envVars := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Parse KEY=VALUE or KEY="VALUE"
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])

		// Remove quotes if present
		if len(value) >= 2 {
			if (value[0] == '"' && value[len(value)-1] == '"') ||
				(value[0] == '\'' && value[len(value)-1] == '\'') {
				value = value[1 : len(value)-1]
			}
		}

		envVars[key] = value
	}

	return envVars, scanner.Err()
}
