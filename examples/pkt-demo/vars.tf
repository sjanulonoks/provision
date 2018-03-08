
//
// PACKET CREDENTIALS
//

// please keep the the API Key and Project ID on single lines for automated cleanup

// Your Packet API Key, grab one from the portal at https://app.packet.net/portal#/api-keys
// NOTE:  THIS WILL BE MODIFIED FOR YOU !!
variable "packet_api_key" { default = "insert_api_key_here" }

// Your Project ID, you can see it here https://app.packet.net/portal#/projects/list/table
// NOTE:  THIS WILL BE MODIFIED FOR YOU !! DO NOT change the formatting of the below line
variable "packet_project_id" { default = "insert_project_id_here" }

//
// GLOBAL - setup values
//

provider "packet" {
       version = "~> 1.0"
    auth_token = "${var.packet_api_key}"
}

// The name of your cluster 
//
// WARNING: no spaces!  Use only dashes as special characters.
//          Preference is to NOT use any dashes, just a short 3 to 8
//          character prefix for the cluster name.  This will be
//          combined to create unique hostnames.
//
// DO NOT change the formatting of the below line !!
variable "cluster_name" { default = "demo" }

// set your Packet billing cycle 
variable "billing_cycle" {
  default = "hourly"
}

//
// INFRASTRUCTURE - DRP Endpoint server parameters
//

// The Packet data center you would like to deploy into
variable "packet_facility" {
  //default = "sjc1"  // Sunnyvale, CA USA
  //default = "nrt1"  // Tokyo, JP
  default = "ewr1"  // Parsipany, NJ USA
}

// The path to the private key you created
variable "drp_ssh_key_path" {
  default = "drp-ssh-key"
}

// The path to the public key you created
variable "drp_ssh_public_key_path" {
  default = "drp-ssh-key.pub"
}
 
// the Digital Rebar Provisioning server Operating System
variable "drp_os" {
  default = "centos_7"
//  default = "ubuntu_16_04"
}

// The Packet DRP endpoint type to use
variable "endpoint_type" {
  default = "baremetal_0"
}


//
// INFRASTRUCTURE - DRP Deployed Machines parameters
//

// The number of Packet machines to have DRP deploy
variable "machines_count" {
  // default = "10"
  default = "1"
}

// The Packet DRP provisioned machines types to use
variable "machines_type" {
  default = "baremetal_0"
}

// NOTE ON the Digital Rebar Provisioning server Operating System
// Provisioned OS is defined by the DRP endpoint itself - and not
// provisioned by the Terraform/Packet provider/plugins.  
variable "machines_os" { default = "centos_7" }
//variable "machines_os" { default = "ubuntu_16_04" }

// The path to the private key you created
variable "machines_ssh_key_path" {
  default = "machines-ssh-key"
}

// The path to the public key you created
variable "machines_ssh_public_key_path" {
  default = "machines-ssh-key.pub"
}
 
