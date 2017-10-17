
//
// Defines the terraform resources for the Digital Rebar Provision
// endpoint itself.  
//

resource "packet_ssh_key" "drp-ssh-key" {
  name       = "${var.cluster_name}-drp-ssh-key"
  public_key = "${file("${var.cluster_name}-${var.drp_ssh_public_key_path}")}"
}

resource "packet_device" "drp-endpoint" {
  hostname         = "${format("${var.cluster_name}-drp-${var.packet_facility}-%02d", count.index)}"
  operating_system = "${var.drp_os}"
  plan             = "${var.endpoint_type}"
  facility         = "${var.packet_facility}"
  project_id       = "${var.packet_project_id}"
  billing_cycle    = "hourly"
}


