#!/bin/bash

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

function confirm() {
  local _sep="------------------------------------------------------------"
  local _wait
  echo ""
  echo "$_sep"
  echo "ACTION: $*"
  echo -n "Run next step?  [ <Enter> | No | Ctrl-C ]  "
  read _wait

  if [[ "$_wait" =~ [Nn].* ]] 
  then
    echo "Skipping..."
    echo "$_sep"
    return
  else
    echo "$_sep"
    echo ""
    $*
  fi
}

#echo "ACTION: Cloning GIT repo contents ... "
#git clone <something>
#cd 5min-drp

#
#vim private-content/secrets
# cp ../private-content/* private-content/

# installs terraform locally
confirm ./control.sh install-terraform    

# installs API and PROJECT secrets for Terraform files
confirm ./control.sh install-secrets      

# removes ssh keys if exists and generates new keys
confirm ./control.sh ssh-keys             

# create our DRP endpoint instance
confirm terraform apply                   
# view our completed plan status
confirm terraform plan                    

# installs DRP locally for CLI commands
confirm ./control.sh get-drp-local        

# get the DRP endpoint server ID
confirm ./control.sh get-drp-id           

# assign our ID to DRP variable for easy reuse below
confirm export DRP=`./control.sh get-drp-id`

# install DRP and basic content as identified by <ID>
confirm ./control.sh drp-install $DRP     
# check that DRP is running
./control.sh ssh $DRP "ps -ef | grep dr-provision"

case $1 in 
  local)
    echo "Installing content to DRP endpoint ('$DRP') from local system (push to endpoint)..."
    # installs DRP community content locally
    confirm ./control.sh get-drp-content      
    # installs DRP Packet Plugins
    confirm ./control.sh get-drp-plugins      
    # perform content and plugins setup on <ID> endpoint
    confirm ./control.sh drp-setup $DRP
  ;;
  remote|*)
    echo "Installing content from DRP endpoint ('$DRP') (pull from endpoint)..."
    # runs 'get-drp-content', 'get-drp-plugins', and 'drp-setup' on remote <ID>
    confirm ./control.sh remote-content $DRP  
  ;;
esac

# helper functions ... not used in demo
#./control.sh get-address <ID>     # get the IP address of new DRP server identified by <ID>
#./control.sh ssh <ID> [COMMANDS]  # ssh to the IP address of DRP server identified by <ID>
#./control.sh scp <ID> [FILES]     # ssh to the IP address of DRP server identified by <ID>



