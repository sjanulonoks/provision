
//provider "packet" {
//    auth_token = "${var.packet_api_key}"
//}

resource "packet_ssh_key" "5min-drp" {
  name       = "5min-drp-demo-key"
  public_key = "${file("${var.drp_ssh_public_key_path}")}"
}

resource "packet_device" "5min-drp" {
  hostname         = "${format("drp-${var.packet_facility}-%02d", count.index)}"
  operating_system = "centos_7"
  plan             = "${var.endpoint_type}"
  facility         = "${var.packet_facility}"
  project_id       = "${var.packet_project_id}"
  billing_cycle    = "hourly"
}


