
resource "packet_ssh_key" "5min-drp-ssh-key" {
  name       = "5min-drp-ssh-key"
  public_key = "${file("${var.drp_ssh_public_key_path}")}"
}

resource "packet_device" "5min-drp" {
  hostname         = "${format("5min-drp-${var.packet_facility}-%02d", count.index)}"
  operating_system = "centos_7"
  plan             = "${var.endpoint_type}"
  facility         = "${var.packet_facility}"
  project_id       = "${var.packet_project_id}"
  billing_cycle    = "hourly"
}


