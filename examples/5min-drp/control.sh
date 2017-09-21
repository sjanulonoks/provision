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

###
#  simple control script to build a DRP endpoint via CLI on a remote
#  target platform ... 
#
#  usage:   $0 --help
#
#   TODO:  * need to use correct URLs for getting content from official
#            repo locations
#          * fix private content get so that we don't have to pre-stage it
###

###
#  exit with error code and FATAL message
#  example:  xiterr 1 "error message"
###
function xiterr() { [[ $1 =~ ^[0-9]+$ ]] && { local _xit=$1; shift; } || local _xit=255; echo "FATAL: $*"; exit $_xit; }

# get our API_KEY and PROJECT_ID secrets
source ./private-content/secrets || xiterr 1 "unable to source './secrets' file "
[[ -z "$API_KEY" ]] && xiterr 1 "API_KEY is empty ... bailing - check secrets file"
[[ -z "$PROJECT_ID" ]] && xiterr 1 "PROJECT_ID is empty ... bailing - check secrets file"

SSH_KEY=${SSH_KEY:-"5min-drp-ssh-key"}
MY_OS=${MY_OS:-"darwin"}
MY_ARCH=${MY_ARCH:-"amd64"}
TF_VER=${TF_VER:-"0.10.5"}
DRP_OS=${DRP_OS:-"linux"}
DRP_ARCH=${DRP_ARCH:-"amd64"}
CREDS=${CREDS:-"--username=rocketskates --password=r0cketsk8ts"}
NODE_OS=${NODE_OS:-"ce-centos-7.3.1611"}  # ce-ubuntu-16.04

CURL="curl -sfSL"
DRPCLI="./drpcli"

# add HOME/bin to path if it's not there already
[[ ":$PATH:" != *":$HOME/bin:"* ]] && PATH="$HOME/bin:${PATH}"

function usage() {

cat <<END_USAGE
USAGE:  $0 [arguments]
WHERE: arguments are as follows:

        install-terraform    installs terraform locally
        install-secrets      installs API and PROJECT secrets for Terraform files
        ssh-keys             removes ssh keys if exists and generates new keys
        get-drp-local        installs DRP locally
        get-drp-content      installs DRP community content locally
        get-drp-plugins      installs DRP Packet Plugins
        remote-content <ID>  runs 'get-drp-content' and 'get-drp-plugins' on remote <ID>
        get-drp-id           get the DRP endpoint server ID
        get-address <ID>     get the IP address of new DRP server identified by <ID>
        ssh <ID> [COMMANDS]  ssh to the IP address of DRP server identified by <ID>
        scp <ID> [FILES]     ssh to the IP address of DRP server identified by <ID>
        drp-install <ID>     install DRP and basic content as identified by <ID>
        drp-setup <ID>       perform content and plugins setup on <ID> endpoint
        cleanup              WARNING WARNING WARNING

CLEANUP:  WARNING - cleanup will NUKE things - like your private SSH KEY (and more) !!!!!!!!!

  NOTES:  * 'get-drp-content' and 'get-drp-plugins' run on the local control host
            'remote-content' runs the content pull FROM the <ID> endpoint
            ONLY run 'get-drp-*' _OR_ 'remote-content' - NOT both

          * get-drp-id gets the ID of the DRP endpoint server - suggest adding
            to your environment varialbes like:
               #  export DRP=\`$0 get-drp-id\`

          * <ID>  is the ID of the DRP endpoint that is created by terraform 

          * you can override built in defaults by setting the following variables:
               SSH_KEY  MY_OS  MY_ARCH  TF_VER  DRP_OS  DRP_ARCH  CREDS  NODE_OS

END_USAGE
} # end usaage()

###
#  accept as ARGv1 a sha256 check sum file to test
###
function check_sum() {
  local _sum=$1
  [[ -z "$_sum" ]] && xiterr 1 "no check sum file passed to check_sum()"
  [[ ! -r "$_sum" ]] && xiterr 1 "unable to read check sum file '$_sum'"
  local _platform=`uname -s`

  case $_platform in
    Darwin) shasum -a 256 -c $_sum ;;
    Linux)  sha256sum -c $_sum ;;
    *) xiterr 2 "unsupported platform type '$_platform'"
  esac
}

function prereqs() {
  local _pkgs
  local _yq="https://gist.githubusercontent.com/earonesty/1d7cb531bb8fff8c228b7710126bcc33/raw/e250f65764c448fe4073a746c4da639d857c9e6c/yq"
  # test for our prerequisites here and add them to _pkgs parameter if missing
  mkdir -p $HOME/bin
  ( which unzip > /dev/null 2>&1 ) || _pkgs="unzip $_pkgs"
  ( which jq > /dev/null 2>&1 ) || _pkgs="jq $_pkgs"
  ( which yq > /dev/null 2>&1 ) || { $CURL $_yq -o $HOME/bin/yq; chmod 755 $HOME/bin/yq; }

  [[ -z "$_pkgs" ]] && return
	os_info

	case $_OS_FAMILY in
		rhel)   sudo yum -y install $_pkgs ;;
		debian) sudo apt -y install $_pkgs ;;
    darwin) ;;
    *)  xiterr 4 "unsupported _OS_FAMILY ('$_OS_FAMILY') in prereqs()" ;;
	esac

}

# set our global _OS_* variables for re-use
function os_info() {
	# Figure out what Linux distro we are running on.
	# set these globally for use outside of the script
	export _OS_TYPE= _OS_VER= _OS_NAME= _OS_FAMITLY=

	if [[ -f /etc/os-release ]]; then
    source /etc/os-release
    _OS_TYPE=${ID,,}
    _OS_VER=${VERSION_ID,,}
	elif [[ -f /etc/lsb-release ]]; then
    source /etc/lsb-release
    _OS_VER=${DISTRIB_RELEASE,,}
    _OS_TYPE=${DISTRIB_ID,,}
	elif [[ -f /etc/centos-release || -f /etc/fedora-release || -f /etc/redhat-release ]]; then
    for rel in centos-release fedora-release redhat-release; do
        [[ -f /etc/$rel ]] || continue
        _OS_TYPE=${rel%%-*}
        _OS_VER="$(egrep -o '[0-9.]+' "/etc/$rel")"
        break
    done

    if [[ ! $_OS_TYPE ]]; then
        echo "Cannot determine Linux version we are running on!"
        exit 1
    fi
	elif [[ -f /etc/debian_version ]]; then
    _OS_TYPE=debian
    _OS_VER=$(cat /etc/debian_version)
	elif [[ $(uname -s) == Darwin ]] ; then
    _OS_TYPE=darwin
    _OS_VER=$(sw_vers | grep ProductVersion | awk '{ print $2 }')
	fi
	_OS_NAME="$_OS_TYPE-$_OS_VER"

	case $_OS_TYPE in
    centos|redhat|fedora) _OS_FAMILY="rhel";;
    debian|ubuntu) _OS_FAMILY="debian";;
    *) _OS_FAMILY=$_OS_TYPE;;
	esac
} # end os_family()

prereqs 

case $1 in 
  usage|--usage|help|--help|-h)
    usage
    ;;

  install-secrets)
      sed -i.bak                                    \
        -e "s/insert_api_key_here/$API_KEY/g"       \
        -e "s/insert_project_id_here/$PROJECT_ID/g" \
        vars.tf
    ;;

  get-drp-local)
    rm -rf dr-provision-install
    mkdir dr-provision-install
    cd dr-provision-install
    $CURL https://github.com/digitalrebar/provision/releases/download/tip/dr-provision.zip -o dr-provision.zip
    $CURL https://github.com/digitalrebar/provision/releases/download/tip/dr-provision.sha256 -o dr-provision.sha256
    check_sum dr-provision.sha256

    unzip dr-provision.zip
    cd ..

    ln -s `pwd`/dr-provision-install/bin/${MY_OS}/${MY_ARCH}/drpcli `pwd`/drpcli
    $DRPCLI version || xiterr 1 "failed to install DRP endpoint in current directory"
    ;;

  install-terraform)
    # get, and install terraform
    cd /tmp
    mkdir -p $HOME/bin
    wget -O tf.zip https://releases.hashicorp.com/terraform/${TF_VER}/terraform_${TF_VER}_${MY_OS}_${MY_ARCH}.zip && unzip tf.zip
    mv terraform $HOME/bin/ && chmod 755 $HOME/bin/terraform
    rm tf.zip

    cd -
    terraform init
    ;;

  get-drp-content)
    echo "No content being added, exists already ... "
#    rm -rf dr-provision-install/drp-community-content.*
#    mkdir -p dr-provision-install
#    cd dr-provision-install

    # community contents
    # it appears it's distributed by default now ... 
#    $CURL \
#      https://github.com/digitalrebar/provision-content/releases/download/tip/drp-community-content.yaml \
#      -o drp-community-content.yaml
#    $CURL \
#      https://github.com/digitalrebar/provision-content/releases/download/tip/drp-community-content.sha256 \
#      -o drp-community-content.sha256
#
#    check_sum drp-community-content.sha256
#    cd ..

    ;;

  get-drp-plugins)
    [[ ! -r private-content/drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.zip ]] && xiterr 1 "missing private-content plugins"

    rm -rf dr-provision-install/drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.*
    mkdir -p dr-provision-install
    cd dr-provision-install

    # packet helper content
    $CURL \
      https://qww9e4paf1.execute-api.us-west-2.amazonaws.com/main/catalog/content/packet \
      -o drp-content-packet.json
    ls -l drp-content-packet.json

# currently these plugins are closed to community - so you MUST obtain this
# with authenticated gitlab account, and copy to the private-content directory
#    $CURL \
#      https://github.com/rackn/provision-plugins/releases/download/tip/drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.sha256 \
#      -o drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.sha256
#    $CURL  \
#      https://github.com/rackn/provision-plugins/releases/download/tip/drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.zip \
#      -o drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.zip
    cp ../private-content/drp-rack-plugins* ./
    check_sum drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.sha256

		rm -rf plugins
    mkdir -p plugins
		cd plugins
		unzip ../drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.zip
    check_sum sha256sums

    cd ../..

    # (drpcli plugins set packet-ipmi parameter packet-api-key API-KEY) 
    ;;

  remote-content)
    [[ -z "$2" ]] && xiterr 1 "Need DRP endpoint ID as argument 2"
    ADDR=`$0 get-address $2`

    $0 ssh $2 "hostname; $0 get-drp-content $2; $0 get-drp-plugins $2; bash -x $0 drp-setup $2"

    ;;

  ssh-keys)
    # remove keys if they exist already 
    [[ -f "${SSH_KEY}" ]] && rm -f ${SSH_KEY}
    [[ -f "${SSH_KEY}.pub" ]] && rm -f ${SSH_KEY}.pub

    ssh-keygen -t rsa -b 4096 -C "5min-DRP-demo" -P "" -f ${SSH_KEY}
    ;;

  get-drp-id)
    terraform plan | grep packet_device.5min-drp | awk ' { print $NF } ' | sed 's/)//'
    ;;

  get-address)
    [[ -z "$2" ]] && xiterr 1 "Need DRP endpoint ID as argument 2"

    [[ ! -r terraform.tfstate ]] && xiterr 3 "terraform.tfstate not readable, did you run 'terraform apply'?"
    cat terraform.tfstate \
      | jq -r '.modules[].resources."packet_device.5min-drp".primary.attributes."network.0.address"'
#    $CURL -X GET --header "Accept: application/json" \
#      --header "X-Auth-Token: ${API_KEY}"              \
#      "https://api.packet.net/devices/${2}"            \
#      | jq -rcM '.ip_addresses[0].address'

    ;;

  ssh|scp)
    CMD=$1
    [[ -z "$2" ]] && xiterr 1 "Need DRP endpoint ID as argument 2"
    TARGET=$2
    shift 2

    case $CMD in
      ssh) ssh -x -i ${SSH_KEY} root@`$0 get-address $TARGET` "$*"
        ;;
      scp) scp -i ${SSH_KEY} $* root@`$0 get-address $TARGET`:
        ;;
    esac
    ;;

  drp-install)
    [[ -z "$2" ]] && xiterr 1 "Need DRP endpoint ID as argument 2"
    A=`$0 get-address $2`

    echo "Pushing helper content to remote DRP endpoint ... "
    echo "           ID :: '$2'"
    echo "   IP Address :: '$A'"
    scp -r -i ${SSH_KEY} -r drp-install.sh terraform.tfstate $0 private-content/ root@${A}:./

    echo "Installing DRP endpoint service on remote host ... "
    ssh -x -i ${SSH_KEY} root@${A} "chmod 755 drp-install.sh; ./drp-install.sh"
    ;;

  drp-setup)
    _ext=""
    [[ -z "$2" ]] && xiterr 1 "Need DRP endpoint ID as argument 2"
    ADDR=`$0 get-address $2`

    ENDPOINT="--endpoint=https://$ADDR:8092 $CREDS"

    # get content
    URLS="
    https://qww9e4paf1.execute-api.us-west-2.amazonaws.com/main/catalog/content/os-linux
    https://qww9e4paf1.execute-api.us-west-2.amazonaws.com/main/catalog/content/os-discovery
    https://qww9e4paf1.execute-api.us-west-2.amazonaws.com/main/catalog/content/packet
    "
    for URL in $URLS
    do
      CONTENT_NAME="content-`basename $URL`.json"
      set -x
      $CURL $URL -o dr-provision-install/$CONTENT_NAME
      set +x
    done

    # install content 
    for CONTENT in dr-provision-install/*content*.[jy][sa]*
    do
      _ext=${CONTENT#*.}
      case $_ext in 
        json)
          CONTENT_NAME=`cat $CONTENT | jq -r '.meta.Name'`
          ;;
        yaml)
          CONTENT_NAME=`cat $CONTENT | yq -r '.meta.Name'`
          ;;
      esac

      if ( $DRPCLI $ENDPOINT contents exists "$CONTENT_NAME" > /dev/null 2>&1 )
      then
        set -x
        $DRPCLI $ENDPOINT contents destroy "$CONTENT_NAME"
        set +x
      fi

      set -x
      $DRPCLI $ENDPOINT contents create - < $CONTENT
      set +x
    done  

    # install paccket-ipmi plugin
    for PLUGIN in dr-provision-install/plugins/packet-ipmi
    do
      PLUG_NAME=`basename $PLUGIN`

      if ( $DRPCLI $ENDPOINT plugin_providers exists $PLUG_NAME > /dev/null 2>&1 )
      then
        set -x
        $DRPCLI $ENDPOINT plugin_providers destroy $PLUG_NAME
        set +x
      fi

      set -x
      $DRPCLI $ENDPOINT plugin_providers upload $PLUGIN as $PLUG_NAME
      set +x
    done

    if ( $DRPCLI $ENDPOINT plugins exists packet-ipmi > /dev/null 2>&1 )
    then
      set -x
      $DRPCLI $ENDPOINT plugins destroy packet-ipmi
      set +x
    fi

    cat <<EOFPLUGIN > private-content/packet-ipmi-plugin-create.json
    {
      "Available": true,
      "Name": "packet-ipmi",
      "Description": "5min Packet IPMI API Key",
      "Provider": "packet-ipmi",
      "Params": { "packet/api-key": "$API_KEY" }
    }
EOFPLUGIN

    if ( $DRPCLI $ENDPOINT plugins exists "packet-ipmi" )
    then
      $DRPCLI $ENDPOINT plugins destroy "packet-ipmi"
    fi
    $DRPCLI $ENDPOINT plugins create - < private-content/packet-ipmi-plugin-create.json
    # set up the packet stage map 
    # create stagemap JSON (NODE_OS:  ubuntu-16.04-install)
	  cat <<EOFSTAGE > private-content/stagemap-create.json
    {
      "Available": true,
      "Description": "5min-packet-map",
      "Name": "global",
      "Params": {
          "change-stage/map": {
            "discover": "packet-discover:Success",
            "packet-discover": "${NODE_OS}:Reboot",
            "packet-ssh-keys": "complete-nowait:Success",
            "${NODE_OS}": "packet-ssh-keys:Success"
        }
      }
    }
EOFSTAGE

    if ( $DRPCLI $ENDPOINT profiles exists global )
    then
      $DRPCLI $ENDPOINT profiles destroy global
    fi
    $DRPCLI $ENDPOINT profiles create - < private-content/stagemap-create.json

    # set our default Stage, Default Boot Enviornment, and our Unknown Boot Environment
    ( $DRPCLI $ENDPOINT stages exists discover ) || xiterr 9 "default stage ('discover') doesn't exist"
    ( $DRPCLI $ENDPOINT bootenvs exists sledgehammer ) || xiterr 9 "default BootEnv ('sledgehammer') doesn't exist"
    ( $DRPCLI $ENDPOINT bootenvs exists discovery ) || xiterr 9 "unknown BootEnv ('discovery') doesn't exist"

    $DRPCLI $ENDPOINT prefs set defaultStage discover defaultBootEnv sledgehammer unknownBootEnv discovery
    ;;

  cleanup)
    ###
    #  brain dead cleanup script ... I hope you know what you're doing ...
    ###
    set -x
    rm -f 5min-drp-ssh-key
    rm -f 5min-drp-ssh-key.pub
    rm -f drpcli
    rm -rf dr-provision-install

    sed -i.bak                                               \
      -e 's/^\(API="\)\(.*\)\("\)/\1insert_api_key_here\3/g'        \
      -e 's/^\(PROJECT="\)\(.*\)\("\)/\1insert_project_id_here\3/g' \
      private-content/secrets
    sed -i.bak                                                                \
      -e 's/\(^.*packet_api_key.*"\)\(.*\)\(".*$\)/\1insert_api_key_here\3/g' \
      -e 's/\(^.*project_id.*"\)\(.*\)\(".*$\)/\1insert_project_id_here\3/g'  \
      vars.tf

    find private-content/ -type f | grep -v "/secrets$" | xargs rm -rf 
    set +x
    echo "No terraform actions taken - please nuke resources via terraform ... "
    echo "Suggest:    terraform destroy"
    echo "            rm -rf .terraform"
    ;;

  *) 
    $0 usage
    xiterr 99 "unknown option(s) '$*'"
    ;;
esac

exit 0
