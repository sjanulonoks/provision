//
// we define machines_data here as we build this dynamically via the control.sh script
// and since Terraform isn't capable enought to inject a variable into a variable 
// block, we have to hard-wire our DRP Endpoint in to this
//

// NOTICE NOTICE NOTICE 
//
// we rely on the drp-endpoint being fully provisioned first, and the IP Address
// and port of the endpoint will be injected in to this file, replacing the 
// 'drp_endpoint_address_and_port' value in the "machines_data" variable below.
///

resource "packet_ssh_key" "machines-ssh-key" {
  name       = "${var.cluster_name}-machines-ssh-key"
  public_key = "${file("${var.cluster_name}-${var.machines_ssh_public_key_path}")}"
}

variable "machines_data" {
default =<<EOF
#!ipxe
chain http://drp_endpoint_address_and_port/default.ipxe
EOF
// example: chain http://147.75.108.41:8091/default.ipxe
}

resource "packet_device" "drp-machines" {
  hostname         = "${format("${var.cluster_name}-machines-${var.packet_facility}-%02d", count.index + 1)}"
  operating_system = "custom_ipxe"
  always_pxe       = "true"
  count            = "${var.machines_count}"
  plan             = "${var.machines_type}"
  facility         = "${var.packet_facility}"
  project_id       = "${var.packet_project_id}"
  billing_cycle    = "${var.billing_cycle}"
  user_data        = "${var.machines_data}"
}

