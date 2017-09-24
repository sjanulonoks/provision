
//
// PACKET CREDENTIALS
//

// please keep the the API Key and Project ID on single lines for automated cleanup

// Your Packet API Key, grab one from the portal at https://app.packet.net/portal#/api-keys
variable "packet_api_key" { default = "insert_api_key_here" }

// Your Project ID, you can see it here https://app.packet.net/portal#/projects/list/table
variable "packet_project_id" { default = "insert_project_id_here" }

//
// GLOBAL - setup values
//

provider "packet" {
    auth_token = "${var.packet_api_key}"
}

//
// INFRASTRUCTURE - DRP Endpoint server
//

// The Packet data center you would like to deploy into
variable "packet_facility" {
  //default = "sjc1"
  default = "nrt1"
}

// The path to the private key you created
variable "drp_ssh_key_path" {
  default = "./5min-drp-ssh-key"
}

// The path to the public key you created
variable "drp_ssh_public_key_path" {
  default = "./5min-drp-ssh-key.pub"
}
 
// The Packet DRP endpoint type to use
variable "endpoint_type" {
  default = "baremetal_0"
}

//
// INFRASTRUCTURE - DRP Deployed Nodes 
//

// The number of Packet nodes to have DRP deploy
variable "cluster_name" {
  default = "5min"
}

// The number of Packet nodes to have DRP deploy
variable "node_count" {
  // default = "10"
  default = "1"
}

// The Packet DRP provisioned node types to use
variable "node_type" {
  default = "baremetal_0"
}

// The path to the private key you created
variable "nodes_ssh_key_path" {
  //default = "./5min-drp-ssh-key"
  default = "./5min-nodes-ssh-key"
}

// The path to the public key you created
variable "nodes_ssh_public_key_path" {
  //default = "./5min-drp-ssh-key.pub"
  default = "./5min-nodes-ssh-key.pub"
}
 

