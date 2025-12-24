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

# TODO: Update for libvirt provider 0.9+ schema
# In 0.9+, network_interface is part of devices.interfaces
# Need to determine correct attribute path for IP addresses
# output "vm_ip" {
#   description = "IP address of the VM"
#   value       = libvirt_domain.vm.network_interface[0].addresses
# }

# output "ssh_command" {
#   description = "SSH command to connect to the VM"
#   value       = "ssh ${var.ssh_user}@${try(libvirt_domain.vm.network_interface[0].addresses[0], "pending")}"
# }

output "console_command" {
  description = "Command to access VM console"
  value       = "virsh console ${var.vm_name}"
}

output "vnc_port" {
  description = "VNC port for graphical console"
  value       = "Run: virsh vncdisplay ${var.vm_name}"
}
