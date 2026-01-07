package create

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCPUOptions(t *testing.T) {
	// Verify we have the expected options
	assert.Len(t, CPUOptions, 3)
	assert.Equal(t, "1 CPU", CPUOptions[0].Label)
	assert.Equal(t, 1, CPUOptions[0].Value)
	assert.Equal(t, "2 CPUs (recommended)", CPUOptions[1].Label)
	assert.Equal(t, 2, CPUOptions[1].Value)
	assert.Equal(t, "4 CPUs", CPUOptions[2].Label)
	assert.Equal(t, 4, CPUOptions[2].Value)
}

func TestMemoryOptions(t *testing.T) {
	assert.Len(t, MemoryOptions, 3)
	assert.Equal(t, "2 GB", MemoryOptions[0].Label)
	assert.Equal(t, 2048, MemoryOptions[0].Value)
	assert.Equal(t, "4 GB (recommended)", MemoryOptions[1].Label)
	assert.Equal(t, 4096, MemoryOptions[1].Value)
	assert.Equal(t, "8 GB", MemoryOptions[2].Label)
	assert.Equal(t, 8192, MemoryOptions[2].Value)
}

func TestDiskOptions(t *testing.T) {
	assert.Len(t, DiskOptions, 3)
	assert.Equal(t, "10 GB", DiskOptions[0].Label)
	assert.Equal(t, 10, DiskOptions[0].Value)
	assert.Equal(t, "20 GB (recommended)", DiskOptions[1].Label)
	assert.Equal(t, 20, DiskOptions[1].Value)
	assert.Equal(t, "40 GB", DiskOptions[2].Label)
	assert.Equal(t, 40, DiskOptions[2].Value)
}

func TestGetCPULabels(t *testing.T) {
	labels := GetCPULabels()
	assert.Len(t, labels, 3)
	assert.Equal(t, "1 CPU", labels[0])
	assert.Equal(t, "2 CPUs (recommended)", labels[1])
	assert.Equal(t, "4 CPUs", labels[2])
}

func TestGetMemoryLabels(t *testing.T) {
	labels := GetMemoryLabels()
	assert.Len(t, labels, 3)
	assert.Equal(t, "2 GB", labels[0])
	assert.Equal(t, "4 GB (recommended)", labels[1])
	assert.Equal(t, "8 GB", labels[2])
}

func TestGetDiskLabels(t *testing.T) {
	labels := GetDiskLabels()
	assert.Len(t, labels, 3)
	assert.Equal(t, "10 GB", labels[0])
	assert.Equal(t, "20 GB (recommended)", labels[1])
	assert.Equal(t, "40 GB", labels[2])
}

func TestGetCPUValue(t *testing.T) {
	// Valid indices
	assert.Equal(t, 1, GetCPUValue(0))
	assert.Equal(t, 2, GetCPUValue(1))
	assert.Equal(t, 4, GetCPUValue(2))

	// Invalid indices return recommended (index 1)
	assert.Equal(t, 2, GetCPUValue(-1))
	assert.Equal(t, 2, GetCPUValue(99))
}

func TestGetMemoryValue(t *testing.T) {
	// Valid indices
	assert.Equal(t, 2048, GetMemoryValue(0))
	assert.Equal(t, 4096, GetMemoryValue(1))
	assert.Equal(t, 8192, GetMemoryValue(2))

	// Invalid indices return recommended (index 1)
	assert.Equal(t, 4096, GetMemoryValue(-1))
	assert.Equal(t, 4096, GetMemoryValue(99))
}

func TestGetDiskValue(t *testing.T) {
	// Valid indices
	assert.Equal(t, 10, GetDiskValue(0))
	assert.Equal(t, 20, GetDiskValue(1))
	assert.Equal(t, 40, GetDiskValue(2))

	// Invalid indices return recommended (index 1)
	assert.Equal(t, 20, GetDiskValue(-1))
	assert.Equal(t, 20, GetDiskValue(99))
}

func TestMultipassImageOptions(t *testing.T) {
	assert.Len(t, MultipassImageOptions, 4)
	assert.Equal(t, "24.04 LTS (Noble Numbat)", MultipassImageOptions[0].Label)
	assert.Equal(t, "24.04", MultipassImageOptions[0].Value)
}

func TestGetMultipassImageLabels(t *testing.T) {
	labels := GetMultipassImageLabels()
	assert.Len(t, labels, 4)
	assert.Equal(t, "24.04 LTS (Noble Numbat)", labels[0])
}

func TestGetMultipassImageValue(t *testing.T) {
	assert.Equal(t, "24.04", GetMultipassImageValue(0))
	assert.Equal(t, "22.04", GetMultipassImageValue(1))

	// Invalid indices return first
	assert.Equal(t, "24.04", GetMultipassImageValue(-1))
	assert.Equal(t, "24.04", GetMultipassImageValue(99))
}

func TestStorageOptions(t *testing.T) {
	assert.Len(t, StorageOptions, 3)
	assert.Equal(t, "LVM (recommended)", StorageOptions[0].Label)
	assert.Equal(t, "lvm", StorageOptions[0].Value)
}

func TestGetStorageLabels(t *testing.T) {
	labels := GetStorageLabels()
	assert.Len(t, labels, 3)
	assert.Equal(t, "LVM (recommended)", labels[0])
}

func TestGetStorageValue(t *testing.T) {
	assert.Equal(t, "lvm", GetStorageValue(0))
	assert.Equal(t, "direct", GetStorageValue(1))
	assert.Equal(t, "zfs", GetStorageValue(2))

	// Invalid indices return LVM
	assert.Equal(t, "lvm", GetStorageValue(-1))
	assert.Equal(t, "lvm", GetStorageValue(99))
}
