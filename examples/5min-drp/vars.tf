
//
// PACKET CREDENTIALS
//

// please keep the the API Key and Project ID on single lines for automated cleanup
// Your Packet API Key, grab one from the portal at https://app.packet.net/portal#/api-keys
variable "packet_api_key" { default = "insert_api_key_here" }
// Your Project ID, you can see it here https://app.packet.net/portal#/projects/list/table
variable "packet_project_id" { default = "insert_project_id_here" }

//
// INFRASTRUCTURE
//

// The Packet data center you would like to deploy into
variable "packet_facility" {
  default = "sjc1"
}

// All server type slugs are available via the API endpoint /plans

// The Packet server type to use
variable "server_type" {
  default = "baremetal_0"
}

// The path to the private key you created
variable "ssh_key_path" {
  default = "./5min-drp-ssh-key"
}

// The path to the public key you created
variable "ssh_public_key_path" {
  default = "./5min-drp-ssh-key.pub"
}
 
