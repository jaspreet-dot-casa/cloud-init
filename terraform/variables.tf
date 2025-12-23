# =============================================================================
# Terraform Variables
# =============================================================================

# =============================================================================
# Libvirt Connection
# =============================================================================

variable "libvirt_uri" {
  description = "Libvirt connection URI"
  type        = string
  default     = "qemu:///system"
}

variable "storage_pool" {
  description = "Libvirt storage pool name"
  type        = string
  default     = "default"
}

variable "network_name" {
  description = "Libvirt network name"
  type        = string
  default     = "default"
}

# =============================================================================
# VM Configuration
# =============================================================================

variable "vm_name" {
  description = "Name of the virtual machine"
  type        = string
  default     = "ubuntu-server"
}

variable "memory_mb" {
  description = "Memory in MB"
  type        = number
  default     = 2048
}

variable "vcpu_count" {
  description = "Number of vCPUs"
  type        = number
  default     = 2
}

variable "disk_size_gb" {
  description = "Disk size in GB"
  type        = number
  default     = 20
}

variable "autostart" {
  description = "Start VM automatically on host boot"
  type        = bool
  default     = false
}

# =============================================================================
# Ubuntu Image
# =============================================================================

variable "ubuntu_image_path" {
  description = "Path to Ubuntu cloud image (qcow2)"
  type        = string
  default     = "/var/lib/libvirt/images/jammy-server-cloudimg-amd64.img"
}

# =============================================================================
# Cloud-Init
# =============================================================================

variable "cloud_init_file" {
  description = "Path to cloud-init user-data file"
  type        = string
  default     = "../cloud-init/cloud-init.yaml"
}
