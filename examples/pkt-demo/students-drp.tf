//
// we define machines_data here as we build this dynamically via the control.sh script
// and since Terraform isn't capable enough to inject a variable into a variable 
// block, we have to hard-wire our DRP Endpoint in to this
//

// default value, usually overriden by command line ENV var
variable students_count { default = "1" }

resource "packet_device" "student-drp" {
  hostname         = "${format("${var.cluster_name}-drp-student%02d", count.index + 1)}"
  operating_system = "custom_ipxe"
  always_pxe       = "true"
  count            = "${var.students_count}"
  plan             = "${var.machines_type}"
  facility         = "${var.packet_facility}"
  project_id       = "${var.packet_project_id}"
  billing_cycle    = "${var.billing_cycle}"
  ipxe_script_url  = "${var.machines_data}"
}

