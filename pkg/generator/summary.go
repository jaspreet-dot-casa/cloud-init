// Package generator provides cloud-init.yaml and summary generation from configuration.
package generator

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jaspreet-dot-casa/cloud-init/pkg/config"
	"github.com/jaspreet-dot-casa/cloud-init/pkg/packages"
)

// escapeMarkdownTable escapes characters that would break markdown table cells.
func escapeMarkdownTable(s string) string {
	// Escape pipe characters which break table cells
	s = strings.ReplaceAll(s, "|", "\\|")
	// Escape newlines which break table rows
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", "")
	return s
}

// escapeMarkdownText escapes characters for general markdown text.
func escapeMarkdownText(s string) string {
	// Escape characters that could be interpreted as markdown
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "*", "\\*")
	s = strings.ReplaceAll(s, "_", "\\_")
	s = strings.ReplaceAll(s, "[", "\\[")
	s = strings.ReplaceAll(s, "]", "\\]")
	s = strings.ReplaceAll(s, "`", "\\`")
	return s
}

// SummaryData holds all data for the summary markdown template.
type SummaryData struct {
	Username   string
	Hostname   string
	FullName   string
	Email      string
	Categories []CategorySummary
	Tailscale  bool
	Docker     bool
}

// CategorySummary represents a category of packages in the summary.
type CategorySummary struct {
	Name     string
	Packages []PackageSummary
}

// PackageSummary represents a single package in the summary.
type PackageSummary struct {
	Name        string
	DisplayName string
	Description string
	Enabled     bool
}

// GenerateSummary generates a markdown summary file from the configuration.
func GenerateSummary(cfg *config.FullConfig, registry *packages.Registry, outputPath string) (err error) {
	// Build summary data
	data := buildSummaryData(cfg, registry)

	// Ensure parent directory exists
	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}

	// Create output file
	f, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create summary file: %w", err)
	}

	// Defer cleanup: close file and remove on error
	defer func() {
		if cerr := f.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("failed to close summary file: %w", cerr)
		}
		if err != nil {
			os.Remove(outputPath)
		}
	}()

	// Render template to file
	// Note: Using = instead of := to avoid shadowing the named return err,
	// which is needed for the deferred cleanup to work correctly.
	if err = Summary(data).Render(context.Background(), f); err != nil {
		return fmt.Errorf("failed to render summary: %w", err)
	}

	return nil
}

// buildSummaryData builds SummaryData from config and registry.
func buildSummaryData(cfg *config.FullConfig, registry *packages.Registry) SummaryData {
	// Create a set of enabled packages for quick lookup
	enabledSet := make(map[string]bool)
	for _, pkg := range cfg.EnabledPackages {
		enabledSet[pkg] = true
	}

	// Build categories with packages (only non-empty categories)
	categories := make([]CategorySummary, 0, len(packages.CategoryOrder))

	for _, cat := range packages.CategoryOrder {
		pkgs, ok := registry.ByCategory[cat]
		if !ok || len(pkgs) == 0 {
			// Skip empty categories - don't allocate what we won't render
			continue
		}

		catSummary := CategorySummary{
			Name:     string(cat),
			Packages: make([]PackageSummary, 0, len(pkgs)),
		}

		for _, pkg := range pkgs {
			catSummary.Packages = append(catSummary.Packages, PackageSummary{
				Name:        pkg.Name,
				DisplayName: escapeMarkdownText(pkg.DisplayName),
				Description: escapeMarkdownText(pkg.Description),
				Enabled:     enabledSet[pkg.Name],
			})
		}

		categories = append(categories, catSummary)
	}

	// Determine service states - check both enabled packages AND config
	// Tailscale is enabled if it's in enabled packages OR has an auth key
	tailscale := enabledSet[packages.PackageTailscale] || cfg.TailscaleAuthKey != ""
	// Docker is enabled if it's in enabled packages
	docker := enabledSet[packages.PackageDocker]

	return SummaryData{
		Username:   escapeMarkdownTable(cfg.Username),
		Hostname:   escapeMarkdownTable(cfg.Hostname),
		FullName:   escapeMarkdownTable(cfg.FullName),
		Email:      escapeMarkdownTable(cfg.Email),
		Categories: categories,
		Tailscale:  tailscale,
		Docker:     docker,
	}
}
