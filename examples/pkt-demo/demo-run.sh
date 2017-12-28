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

################## 
################## SEE README for details on usage of this script .... 
################## 

function xiterr() { [[ $1 =~ ^[0-9]+$ ]] && { local _xit=$1; shift; } || local _xit=255; echo "FATAL: $*"; exit $_xit; }
[[ -f ./bin/color.sh ]] && source ./bin/color.sh
( type -t cprintf > /dev/null 2>&1 ) || function cprintf() { printf "%s" "$*"; }

[[ ":$PATH:" != *":`pwd`/bin:"* ]] && PATH="`pwd`/bin:${PATH}"

cloudia.sh

CONFIRM=${CONFIRM:-"yes"}
SKIP_LOCAL=${SKIP_LOCAL:-"no"}

function sep() {
  local _sep="--------------------------------------------------------------------------------"
  cprintf $magenta "${_sep}\n"
}

function confirm() {
  local _err=0
  local _wait=yes
  local _exit_if_fail=0
  local _action=`cprintf "$bold$underline" "ACTION"`
  local _msg=`cprintf $green "$*"`
  local _default=`cprintf $cyan "<Enter>"`
  local _failed=`cprintf $red "FAILED"`
  local _success=`cprintf $green "Success... "`
  local _skipping=`cprintf $yellow "Skipping..."`

  [[ $1 == "exit_if_fail" ]] && { shift 1; _exit_if_fail=1; }

  echo ""
  sep
  echo "$_action :: $_msg"

  if [[ $CONFIRM == "yes" ]] 
  then
    echo -n "Run next step?  [ $_default | No | Ctrl-C ]  "
    read _wait
  fi

  if [[ "$_wait" =~ [Nn].* ]] 
  then
    echo "$_skipping"
    sep
    return
  else
    sep
    echo ""
    eval $*
    _err=$?

    ((  $_err && $_exit_if_fail )) && xiterr 2 "FAILURE: exiting - '$*'"
    (( $_err )) && echo "$_failed" || echo "$_success"
  fi
}

###
#  we assume you've checked out the examples/pkt-demo/ directory from the
#  Digital Rebar Provision repo ... do something like:
###

# echo "ACTION: Cloning GIT repo contents ... "
# git clone -n https://github.com/digitalrebar/provision.git --depth=1
# cd provision
# git checkout HEAD examples/pkt-demo
# cd ..
# mv examples/pkt-demo $HOME/
# cd $HOME/pkt-demo

if [[ "$USER" == "shane" ]]
then
  echo "<<SHANE>> Staging secrets ... "
  set -x
  cp $HOME/private-content/secrets ./private-content
  set +x
fi

# installs terraform locally
confirm control.sh install-terraform    

# installs API and PROJECT secrets for Terraform files
confirm control.sh install-secrets      

# removes ssh keys if exists and generates new keys
confirm control.sh ssh-keys             

# apply our SSH keys 
confirm exit_if_fail terraform apply -target=packet_ssh_key.drp-ssh-key -auto-approve
confirm exit_if_fail terraform apply -target=packet_ssh_key.machines-ssh-key -auto-approve

# build our DRP server
confirm exit_if_fail terraform apply -target=packet_device.drp-endpoint -auto-approve


# view our completed plan status -- NOTE the "machines"
# do NOT get applied until after 'drp' endpoint is finished 
confirm terraform plan                    

if [[ $SKIP_LOCAL == "no" ]] 
then
  # installs DRP locally for CLI commands
  confirm control.sh get-drp-local        
else
  sep
  _skip_local=`cprintf $green "Skipping 'get-drp-local' as requested... "`
  echo "$_skip_local"
  sep
fi

# get the DRP endpoint server ID
confirm control.sh get-drp-id           

# assign our ID to DRP variable for easy reuse below
confirm export DRP=`control.sh get-drp-id`

# get our DRP Endpoint IP Address to manipulate our SSH Host Keys
confirm export ADDR=`control.sh get-address $DRP`

# remove any existing host keys that might conflict
confirm ssh-keygen -R $ADDR

# scan our newly built host for host keys and inject to known_hosts
confirm "ssh-keyscan -H $ADDR >> $HOME/.ssh/known_hosts"

# install DRP and basic content as identified by <ID>
confirm control.sh drp-install $DRP     

case $1 in 
  local)
    echo "Installing content to DRP endpoint ('$DRP') from local system (push to endpoint)..."
    # runs get-drp-cc, get-drp-plugins, and drp-setup locally
    confirm control.sh local-content-demo $DRP
  ;;
  remote|*)
    echo "Installing content from DRP endpoint ('$DRP') (pull from endpoint)..."
    # runs 'local-content' on remote <DRP>
    echo ""
    cprintf $bold "   SSH to remote DRP, stop, restart in foreground ... ? \n"
    cprintf $bold "   Maybe launch UI to show empty content too ... ? \n"
    cprintf $bold "   https://rackn.github.io/provision-ux/#/e/${ADDR}:8092/system "
    echo ""
    confirm control.sh remote-content-demo $DRP  
    echo ""
    cprintf $cyan "NOTICE:"
    echo "  Errors may be 'normal' - ISOs, Kernel, and InitRDs are "
    echo "         normal as the content has not yet been pushed to the DRP"
    echo "         endpoint.  Other errors should be investigated."
    echo ""
  ;;
esac

# inject our DRP endpoint address in to the drp-machines.tf terraform file
confirm control.sh set-drp-endpoint $DRP

# bug in Stages causes stage "discover" to be marked bad
# the simplest fix is to HUP the dr-provision service
#confirm echo "Restart DRP Endpoint to fix stages bug ?" \
#  && control.sh ssh $DRP 'kill -HUP `pidof dr-provision`'

# bring up our DRP target machines:
confirm terraform apply -target=packet_device.drp-machines -auto-approve

# helper functions ... not used in demo
#control.sh get-address <ID>     # get the IP address of new DRP server identified by <ID>
#control.sh ssh <ID> [COMMANDS]  # ssh to the IP address of DRP server identified by <ID>
#control.sh scp <ID> [FILES]     # ssh to the IP address of DRP server identified by <ID>



