# =============================================================================
# Terraform Outputs
# =============================================================================

output "vm_name" {
  description = "Name of the created VM"
  value       = libvirt_domain.vm.name
}

output "vm_id" {
  description = "ID of the created VM"
  value       = libvirt_domain.vm.id
}

# Note: For libvirt provider 0.9+, the IP may take a moment to be assigned
# The wait_for_ip block in main.tf ensures the IP is available before terraform completes
output "vm_ip" {
  description = "IP address of the VM (from DHCP lease)"
  value       = try(libvirt_domain.vm.devices[0].interfaces[0].addresses[0], "pending")
}

output "ssh_command" {
  description = "SSH command to connect to the VM"
  value       = "ssh ubuntu@${try(libvirt_domain.vm.devices[0].interfaces[0].addresses[0], "pending")}"
}

output "console_command" {
  description = "Command to access VM console"
  value       = "virsh console ${var.vm_name}"
}

output "vnc_port" {
  description = "VNC port for graphical console"
  value       = "Run: virsh vncdisplay ${var.vm_name}"
}

output "vm_running" {
  description = "Whether the VM is running"
  value       = var.running
}

output "vm_memory_mb" {
  description = "Memory allocated to the VM in MB"
  value       = var.memory_mb
}

output "vm_vcpu_count" {
  description = "Number of vCPUs allocated to the VM"
  value       = var.vcpu_count
}

output "vm_disk_size_gb" {
  description = "Disk size in GB"
  value       = var.disk_size_gb
}

output "vm_autostart" {
  description = "Whether the VM autostarts on host boot"
  value       = var.autostart
}
