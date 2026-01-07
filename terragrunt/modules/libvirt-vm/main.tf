# =============================================================================
# OpenTofu/Terraform Module: libvirt VM
#
# Provisions Ubuntu VMs with cloud-init configuration via libvirt/KVM.
# =============================================================================

terraform {
  required_version = ">= 1.0"

  required_providers {
    libvirt = {
      source  = "dmacvicar/libvirt"
      version = "~> 0.9.0"
    }
  }
}

# =============================================================================
# Base Volume (Ubuntu Cloud Image)
# =============================================================================

resource "libvirt_volume" "ubuntu_base" {
  name = "${var.vm_name}-base.qcow2"
  pool = var.storage_pool
  create = {
    content = {
      url = var.ubuntu_image_path
    }
  }
}

# =============================================================================
# VM Volume (Cloned from Base)
# =============================================================================

resource "libvirt_volume" "vm_disk" {
  name     = "${var.vm_name}-disk.qcow2"
  pool     = var.storage_pool
  capacity = var.disk_size_gb * 1024 * 1024 * 1024
  backing_store = {
    path = libvirt_volume.ubuntu_base.path
  }
}

# =============================================================================
# Cloud-Init Configuration
# =============================================================================

resource "libvirt_cloudinit_disk" "cloud_init" {
  name      = "${var.vm_name}-cloudinit.iso"
  user_data = file(var.cloud_init_file)

  meta_data = <<-EOF
    instance-id: ${var.vm_name}
    local-hostname: ${var.vm_name}
  EOF

  network_config = <<-EOF
    version: 2
    ethernets:
      ens3:
        dhcp4: true
  EOF
}

# =============================================================================
# Virtual Machine
# =============================================================================

resource "libvirt_domain" "vm" {
  name        = var.vm_name
  type        = "kvm"
  memory      = var.memory_mb
  memory_unit = "MiB"
  vcpu        = var.vcpu_count
  autostart   = var.autostart
  running     = var.running

  os = {
    type         = "hvm"
    type_arch    = "x86_64"
    type_machine = "q35"
  }

  cpu = {
    mode = "host-passthrough"
  }

  devices = {
    disks = [
      {
        source = {
          volume = {
            pool   = var.storage_pool
            volume = libvirt_volume.vm_disk.name
          }
        }
        target = {
          dev = "vda"
          bus = "virtio"
        }
      },
      {
        source = {
          file = libvirt_cloudinit_disk.cloud_init.id
        }
        target = {
          dev = "vdb"
          bus = "virtio"
        }
        read_only = true
      }
    ]

    interfaces = [
      {
        model = {
          type = "virtio"
        }
        source = {
          network = {
            network = var.network_name
          }
        }
        wait_for_ip = {
          timeout = 300
          source  = "lease"
        }
      }
    ]

    consoles = [
      {
        type = "pty"
        target = {
          type = "serial"
          port = "0"
        }
      }
    ]

    graphics = [
      {
        type = "vnc"
        listen = {
          type = "address"
        }
        auto_port = "yes"
      }
    ]
  }
}
