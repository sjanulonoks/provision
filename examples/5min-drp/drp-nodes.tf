//
// we define node_data here as we build this dynamically via the control.sh script
// and since Terraform isn't capable enought to inject a variable into a variable 
// block, we have to hard-wire our DRP Endpoint in to this
//

resource "packet_ssh_key" "5min-nodes" {
  name       = "5min-nodes-demo-key"
  public_key = "${file("${var.nodes_ssh_public_key_path}")}"
}

variable "node_data" {
default =<<EOF
#!ipxe
echo mac...............: $${mac}
echo ip................: $${ip}
echo netmask...........: $${netmask}
echo gateway...........: $${gateway}
echo dns...............: $${dns}
echo domain............: $${domain}
echo dhcp-server.......: $${dhcp-server}
echo syslog............: $${syslog}
echo filename..........: $${filename}
echo next-server.......: $${next-server}
echo hostname..........: $${hostname}
echo uuid..............: $${uuid}
echo serial............: $${serial}
echo .
chain http://147.75.108.41:8091/default.ipxe
EOF
//kernel $${base-url}/vmlinuz dhcp modprobe.blacklist=ixgbe boot=live fetch=$${base-url}/filesystem.squashfs console=tty0 console=ttyS1,115200
//initrd $${base-url}/initrd.img
//boot
}

resource "packet_device" "5min-nodes" {
  hostname         = "${format("nodes-${var.packet_facility}-${var.cluster_name}-%02d", count.index + 1)}"
  operating_system = "custom_ipxe"
  count            = "${var.node_count}"
  plan             = "${var.node_type}"
  facility         = "${var.packet_facility}"
  project_id       = "${var.packet_project_id}"
  billing_cycle    = "hourly"
  always_pxe       = "true"
  user_data        = "${var.node_data}"
}

