terraform {
  required_providers {
    proxmox = {
      source  = "bpg/proxmox"
      version = "~> 0.111"
    }
  }
}

variable "ssh_public_key_path" {
  type = string
}

variable "snip_ssh_key" {
  type = string
}

variable "proxmox_endpoint" {
  type = string
}

variable "proxmox_api_token" {
  type = string
}

variable "scenarios" {
  type = map(object({
    repo = string 
  }))
  default = {
    box1 = { repo = "https://github.com/range-as-code/helloWorld.git" }
    box2 = { repo = "https://github.com/range-as-code/helloNix.git" }
  }
}

provider "proxmox" {
  insecure = true
  endpoint = var.proxmox_endpoint
  api_token = var.proxmox_api_token
  
  ssh {
    username    = "tofu-snip"
    private_key = file(var.snip_ssh_key)
  }
}

resource "random_password" "vm" {
  for_each = var.scenarios

  length           = 5
  special          = true
  override_special = "!@#$%&*()-_=+[]{}<>:?"
}

resource "proxmox_virtual_environment_file" "user_data" {
  for_each = var.scenarios

  content_type = "snippets"
  datastore_id = "local"
  node_name    = "lab1"

  source_raw {
    data      = <<-EOF
      #cloud-config
      users:
        - name: debian
          sudo: ALL=(ALL) NOPASSWD:ALL
          shell: /bin/bash
          ssh_authorized_keys:
            - ${trimspace(file(var.ssh_public_key_path))}
      ssh_pwauth: true
      chpasswd:
        list: |
          debian:${random_password.vm[each.key].result}
        expire: False
      write_files:
        - path: /etc/scenario-repo
          content: ${each.value.repo}
    EOF
    file_name = "user-data-${each.key}.yaml"
  }
}

resource "proxmox_virtual_environment_vm" "target" {
  for_each = var.scenarios
  name = each.key
  node_name = "lab1"
  clone { vm_id = 9104 }
  agent { enabled = true }

  network_device {
    bridge = "rvnet"
  }

  initialization {
    user_data_file_id = proxmox_virtual_environment_file.user_data[each.key].id
    ip_config { 
      ipv4 { address = "dhcp" } 
      }
  }
}

output "credentials" {
  value = {
    for k, vm in proxmox_virtual_environment_vm.target : k => {
      ip = vm.ipv4_addresses
      username = "debian"
      password = random_password.vm[k].result
    }
  }
  sensitive = true
}
