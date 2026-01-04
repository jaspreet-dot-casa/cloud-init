// Package create provides shared option definitions for the wizard.
package create

// SelectOption represents a single option in a select field.
type SelectOption[T any] struct {
	Label string
	Value T
}

// CPUOptions defines available CPU configurations.
var CPUOptions = []SelectOption[int]{
	{Label: "1 CPU", Value: 1},
	{Label: "2 CPUs (recommended)", Value: 2},
	{Label: "4 CPUs", Value: 4},
}

// MemoryOptions defines available memory configurations in MB.
var MemoryOptions = []SelectOption[int]{
	{Label: "2 GB", Value: 2048},
	{Label: "4 GB (recommended)", Value: 4096},
	{Label: "8 GB", Value: 8192},
}

// DiskOptions defines available disk configurations in GB.
var DiskOptions = []SelectOption[int]{
	{Label: "10 GB", Value: 10},
	{Label: "20 GB (recommended)", Value: 20},
	{Label: "40 GB", Value: 40},
}

// GetCPULabels returns labels for CPU options.
func GetCPULabels() []string {
	return getLabels(CPUOptions)
}

// GetMemoryLabels returns labels for memory options.
func GetMemoryLabels() []string {
	return getLabels(MemoryOptions)
}

// GetDiskLabels returns labels for disk options.
func GetDiskLabels() []string {
	return getLabels(DiskOptions)
}

// getLabels extracts labels from a slice of SelectOptions.
func getLabels[T any](options []SelectOption[T]) []string {
	labels := make([]string, len(options))
	for i, opt := range options {
		labels[i] = opt.Label
	}
	return labels
}

// GetCPUValue returns the CPU value at the given index.
func GetCPUValue(idx int) int {
	if idx < 0 || idx >= len(CPUOptions) {
		return CPUOptions[1].Value // Default to recommended
	}
	return CPUOptions[idx].Value
}

// GetMemoryValue returns the memory value (in MB) at the given index.
func GetMemoryValue(idx int) int {
	if idx < 0 || idx >= len(MemoryOptions) {
		return MemoryOptions[1].Value // Default to recommended
	}
	return MemoryOptions[idx].Value
}

// GetDiskValue returns the disk value (in GB) at the given index.
func GetDiskValue(idx int) int {
	if idx < 0 || idx >= len(DiskOptions) {
		return DiskOptions[1].Value // Default to recommended
	}
	return DiskOptions[idx].Value
}
