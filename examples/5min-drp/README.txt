# Copyright (c) 2017 RackN Inc.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.


README file for 5min-drp


The following are the basic steps 

git clone -n https://github.com/digitalrebar/provision.git --depth=1
cd provision
git checkout HEAD examples/5min-drp
cd examples/5min-drp

# EDIT THE SECRETS FILE !!  You need:
# 
#   API_KEY     packet.net key for access to your Packet Project
#   PROJECT_ID  packet.net project to create DRP and Nodes in 
#
#vim private-content/secrets

# staged demo includes our 'secrets' and RackN private content
# which we'll copy over here to make things easier
cp ../private-content/drp-rack-plugins* ./private-content/
cp ../private-content/terraform-provider-packet bin/
# extra content
cp ../private-content/5min* ./
cp ../private-content/terraform.tfstate ./
cp ../private-content/secrets ./private-content

./control.sh install-terraform    # installs terraform locally
./control.sh install-secrets      # installs API and PROJECT secrets for Terraform files
./control.sh ssh-keys             # removes ssh keys if exists and generates new keys

terraform apply                   # create our DRP endpoint instance
terraform plan                    # view our completed plan status

./control.sh get-drp-local        # installs DRP locally for CLI commands
#./control.sh get-drp-content      # installs DRP community content locally
#./control.sh get-drp-plugins      # installs DRP Packet Plugins
#./control.sh drp-setup <ID>       # perform content and plugins setup on <ID> endpoint

./control.sh get-drp-id           # get the DRP endpoint server ID
export DRP=`./control.sh get-drp-id`
                                  # assign our ID to DRP variable for easy reuse below

./control.sh drp-install $DRP     # install DRP and basic content as identified by <ID>
./control.sh ssh $DRP "ps -ef | grep dr-provision"
                                  # check that DRP is running
./control.sh remote-content $DRP  # runs 'get-drp-content' and 'get-drp-plugins' 
                                  # and drp-setup on remote <ID>


# helper functions ... not used in demo
./control.sh get-address <ID>     # get the IP address of new DRP server identified by <ID>
./control.sh ssh <ID> [COMMANDS]  # ssh to the IP address of DRP server identified by <ID>
./control.sh scp <ID> [FILES]     # ssh to the IP address of DRP server identified by <ID>



