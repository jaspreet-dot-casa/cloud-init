package doctor

import (
	"runtime"
	"sync"
)

// Checker provides dependency checking functionality.
type Checker struct {
	executor   CommandExecutor
	platform   string
	imagePath  string // Path to cloud image for Terraform
}

// NewChecker creates a new Checker with the real command executor.
func NewChecker() *Checker {
	return &Checker{
		executor: &RealExecutor{},
		platform: runtime.GOOS,
	}
}

// NewCheckerWithExecutor creates a new Checker with a custom executor (for testing).
func NewCheckerWithExecutor(exec CommandExecutor) *Checker {
	return &Checker{
		executor: exec,
		platform: runtime.GOOS,
	}
}

// SetImagePath sets the path to check for cloud images.
func (c *Checker) SetImagePath(path string) {
	c.imagePath = path
}

// CheckAll runs all applicable checks and returns groups with results.
func (c *Checker) CheckAll() []CheckGroup {
	groups := GetGroups()
	var result []CheckGroup

	for _, group := range groups {
		checkedGroup := c.CheckGroup(group.ID)
		result = append(result, checkedGroup)
	}

	return result
}

// CheckAllAsync runs all checks concurrently and returns groups with results.
func (c *Checker) CheckAllAsync() []CheckGroup {
	groups := GetGroups()
	result := make([]CheckGroup, len(groups))
	var wg sync.WaitGroup

	for i, group := range groups {
		wg.Add(1)
		go func(idx int, g CheckGroup) {
			defer wg.Done()
			result[idx] = c.CheckGroup(g.ID)
		}(i, group)
	}

	wg.Wait()
	return result
}

// CheckGroup runs all checks for a specific group.
func (c *Checker) CheckGroup(groupID string) CheckGroup {
	def, ok := GetGroupDefinition(groupID)
	if !ok {
		return CheckGroup{
			ID:   groupID,
			Name: "Unknown",
		}
	}

	group := CheckGroup{
		ID:          groupID,
		Name:        def.Name,
		Description: def.Description,
		Platform:    def.Platform,
	}

	for _, checkID := range def.CheckIDs {
		check := c.runCheck(checkID)
		group.Checks = append(group.Checks, check)
	}

	return group
}

// runCheck runs a specific check by ID.
func (c *Checker) runCheck(checkID string) Check {
	switch checkID {
	case IDTerraform:
		return CheckTerraform(c.executor)
	case IDMultipass:
		return CheckMultipass(c.executor)
	case IDXorriso:
		return CheckXorriso(c.executor)
	case IDLibvirt:
		return CheckLibvirt(c.executor)
	case IDVirsh:
		return CheckVirsh(c.executor)
	case IDQemuKVM:
		return CheckQemuKVM(c.executor)
	case IDCloudImage:
		return CheckCloudImage(c.executor, c.imagePath)
	default:
		return Check{
			ID:      checkID,
			Name:    checkID,
			Status:  StatusError,
			Message: "unknown check",
		}
	}
}

// GetCheck runs a single check by ID.
func (c *Checker) GetCheck(checkID string) Check {
	return c.runCheck(checkID)
}

// Summary represents an overall health summary.
type Summary struct {
	Total    int
	OK       int
	Missing  int
	Warnings int
	Errors   int
}

// GetSummary returns a summary of check results.
func (c *Checker) GetSummary(groups []CheckGroup) Summary {
	var summary Summary

	for _, group := range groups {
		for _, check := range group.Checks {
			summary.Total++
			switch check.Status {
			case StatusOK:
				summary.OK++
			case StatusMissing:
				summary.Missing++
			case StatusWarning:
				summary.Warnings++
			case StatusError:
				summary.Errors++
			}
		}
	}

	return summary
}

// HasIssues returns true if any checks have issues.
func (c *Checker) HasIssues(groups []CheckGroup) bool {
	summary := c.GetSummary(groups)
	return summary.Missing > 0 || summary.Errors > 0
}
