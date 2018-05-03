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
TF_VARS=TF_VARS

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
  echo "<<HI SHANE!>> Staging secrets for you ... :)"
  set -x
  cp $HOME/private-content/secrets ./private-content
  set +x
fi


sep
echo ""
echo "Typically set STUDENTS count to 1 greater than number of actual students"
echo "for spares and/or instructor follow-along systems."
echo ""
read -p "              Enter number of students: " STUDENTS
read -p "     Enter target machines per student: " TARGETS
echo ""

[[ $STUDENTS =~ ^[0-9]+$ ]] \
  || xiterr 1 "Non-numeric value ('$STUDENTS') specified for STUDENTS"
[[ $TARGETS =~ ^[0-9]+$ ]] \
  || xiterr 1 "Non-numeric value ('$TARGETS') specified for TARGETS"

(( TOTAL = STUDENTS * ( TARGETS + 1 ) ))

sep

__student_count=`cprintf $green "$STUDENTS"`
__target_count=`cprintf $green "$TARGETS"`
__total_count=`cprintf $green "$TOTAL"`
echo "                    Student count set to:  $__student_count"
echo "Target machines per student count set to:  $__target_count"
echo "             TOTAL machines count set to:  $__total_count"
sep

# dummy run to just check prereqs and set_cluster_name 
confirm control.sh safety_checks

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

# we must do these for all steps - so no confirm on this
# get the DRP endpoint server ID
# assign our ID to DRP variable for easy reuse below
export DRP=`control.sh get-drp-id`
# get our DRP Endpoint IP Address to manipulate our SSH Host Keys
export ADDR=`control.sh get-address $DRP`

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
    cprintf $bold "   https://portal.rackn.io/#/e/${ADDR}:8092/system "
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

# bring up our DRP target machines:
#confirm terraform apply -target=packet_device.drp-machines -auto-approve

# prepare our Student DRP systems - they will be installed with CentOS by
# the default Workflow on the Master DRP 
cat <<EOFTF >> $TF_VARS
export TF_VAR_students_count=$STUDENTS 
EOFTF
source $TF_VARS
confirm exit_if_fail terraform apply -target=packet_device.student-drp -auto-approve

# once the above Student DRP machines are created, we should have assigned
# IP addresses - even if the OS isn't yet built - we now need to grab those
# and start iteratively building the Target Machines
PRE=".modules[0].resources.\"packet_device"
PART="\".primary"
STUDENT=1
ENDPOINT=""
iPXE=""
MAPPING="student-mappings.txt"
TARGETS_TF="student-targets.tf"

# get the students DRP endpoint IP address from the JSON tfstate
function get_student_info() {
  local _type=$1
  local _student=$2
  local _index _idx _num _post _t _tgt _e
  (( _num = _student - 1 ))
  local _nn=$(printf "%02d" "$_student")

#### bloody packet terraform provider enforces ipxe_script_url
#### must exist otherwise it'll fail to provision the machines
####
#  # thank you ... thank you yet again ...

 [[ "$STUDENTS" == "1" ]] && _index="${PRE}.student-drp${PART}" || _index="${PRE}.student-drp.${_num}${PART}"

  case $_type in
    drp|endpoint)
      _post="attributes.access_public_ipv4"
#      '.modules[0].resources."packet_device.0".primary.attributes.access_public_ipv4'
set -x
      jq -r "${_index}.${_post}" terraform.tfstate
set +x
      ;;
    id)
      _t=0
      _tgt=$TARGETS
      while (( _t < _tgt ))
      do
#   jq -r '.modules[0].resources."packet_device.student0-targets.2".primary.id' terraform.tfstate
        _idx="${PRE}.student${_nn}-targets.${_t}${PART}.id"
        _e="`jq -r "${_idx}" terraform.tfstate` $_e"
        (( _t++ ))
      done
      # regurgitate the UUIDs of our targets
      echo "$_e"
      ;;
  esac
}

function get_master_drp() {
  jq -r '.modules[0].resources."packet_device.drp-endpoint".primary.attributes.access_public_ipv4' terraform.tfstate
}

# execute the terraform apply for the given target, we also
# set the student number for the hostname mappings, the
# ipxe_script_url, and the number of target machines to build
# per student - bound to their DRP endpoint
function tf_target_apply() {
  local _s=$(printf "%02d" "$1")
  local _i=$2
  local _m=$3
  cat <<EOFTFVARS >> $TF_VARS
export TF_VAR_student="$_s" 
export TF_VAR_targets_ipxe="$_i" 
export TF_VAR_targets_count="$_m"
EOFTFVARS
  source $TF_VARS
  confirm terraform apply -target=packet_device.student${_s}-targets -auto-approve
}

echo "# run at date: `date`" >> $MAPPING

# write header to TARGETS_TF
cat <<EOFHDR > $TARGETS_TF
###
#  dynamically generated file - do not edit
###

variable student { default = "1" }
variable targets_count { default = "1" }
variable targets_ipxe { default = "" }
EOFHDR

while (( STUDENT <= STUDENTS ))
do
  NN=$(printf "%02d" "$STUDENT")

cat <<EOFST >> $TARGETS_TF

# generated student $NN resource
resource "packet_device" "student$NN-targets" {
  hostname         = "\${format("\${var.cluster_name}-student\${var.student}-target%02d", count.index + 1)}"
  operating_system = "custom_ipxe"
  always_pxe       = "true"
  count            = "\${var.targets_count}"
  plan             = "\${var.machines_type}"
  facility         = "\${var.packet_facility}"
  project_id       = "\${var.packet_project_id}"
  billing_cycle    = "\${var.billing_cycle}"
  ipxe_script_url  = "\${var.targets_ipxe}"
}
EOFST
  (( STUDENT++ ))
done

STUDENT=1
while (( STUDENT <= STUDENTS ))
do
  S_DRP=`get_student_info drp $STUDENT`
  ENDPOINT=`get_master_drp $STUDENT`
  #ENDPOINT="$S_DRP"
  iPXE="http://${ENDPOINT}:8091/default.ipxe"
  tf_target_apply "$STUDENT" "$iPXE" "$TARGETS"

  S_ID=`get_student_info id $STUDENT`

  # generate our mapping file to come back and reset
  # the ipxe_script_urls correctl since the bloody TF
  # provider fails out if the target ipxe_script_url
  # is invalid
#  "packet_device.student02-targets.1"
  for ID in $S_ID
  do
    echo "$ID:$S_DRP" >> $MAPPING
  done

  (( STUDENT++ ))
done

exit 0
