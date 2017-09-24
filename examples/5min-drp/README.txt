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


IMPORTANT REQUIREMENTS
----------------------
You will need to perform the following requirements in preparation to
using this demo 5min-drp process:

  * have a packet.net account, and your API KEY, and PROJECT ID

  * download the RackN registered content pack and stage it in the
    "private-content" directory, along with the sha256sum file

  * download the 'terraform-provider-packet' plugin for terraform
    and stage it in the 'private-content' directory

     + install Go Lang 1.9.0 or newer from:
       https://golang.org/dl/
     + build the provider:
       go get -u github.com/terraform-providers/terraform-provider-packet
     + copy the provider to the 'private-content' directory
       cp $HOME/go/bin/terraform-provider-packet private-content/

  * download RackN "drp-rack-plugin", which is available at:
    https://github.com/rackn/provision-plugins/releases/download/${VER_PLUGINS}/drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.sha256
    https://github.com/rackn/provision-plugins/releases/download/${VER_PLUGINS}/drp-rack-plugins-${DRP_OS}-${DRP_ARCH}.zip


WARNING
-------
Because Terraform has some ... oddities - we must write a $HOME/.terraformrc
file to be able to use the Beta version of the 'terraform-provider-packet'. 

We
will make a backup copy of your existing .terraformrc file - and we will 
restore it at the end of the demo when you run the "control.sh cleanup" 
function.  

IF YOU DO NOT run the 'cleanup' function - you will need to manually restore 
your terraform configuration file. (backup is:  $HOME/.terraform.5min-backup )


GIT CLONE
---------

The following steps will clone this content from the digitalrebar/provision 
github repo:

    git clone -n https://github.com/digitalrebar/provision.git --depth=1
    cd provision
    git checkout HEAD examples/5min-drp
    cd examples/5min-drp
    mv 5min-drp $HOME/
    cd ../..
    rm -rf provision
    cd $HOME/5min-drp

SECRETS INFORMATION
-------------------

# EDIT THE SECRETS FILE !!  You need:
# 
#   API_KEY     packet.net key for access to your Packet Project
#   PROJECT_ID  packet.net project to create DRP and Nodes in 

    vim private-content/secrets


PRIVATE CONTENT
---------------

Make sure you have the RackN registered user content in the 'private-contents'
directory, as follows:

    ls -l private-content/
    -rw-r--r--@ 1      99  Sep 20 11:51  drp-rack-plugins-linux-amd64.sha256
    -rw-r--r--@ 1 9127420  Sep 20 11:51  drp-rack-plugins-linux-amd64.zip
    -rw-r--r--  1     577  Sep 20 11:12  secrets


RUN DEMO-RUN.SH SCRIPT
----------------------

The 'demo-run.sh' is the control script that will walk you through the deployment
process.  Simply start this script. 

    ./demo-run.sh

If you re-run the script and want to skip steps that have run previously, simply 
answer "no" to the "ACTION" input. 

WHAT HAPPENS?
-------------

1.  set PATH to include 5min-drp/bin
2.  install terraform locally in your 5min-drp/bin directory
3.  install the secrets to the terraform vars.tf file
4.  create SSH keys for DRP endpoint and nodes
5.  inject the SSH keys in to packet.net Project
6.  build the DRP Endpoint server in packet.net (on centos-7)
7.  install DRP locally in 5min-drp/bin for 'drpcli' control
8.  install DRP on the remote endpoint  
9.  configure content and services on DPR   [NOTE: CONTENT]
10. set the DRP endpoint IP address to terraform 'drp-nodes.tf'
11. kick over "N" number of nodes to provision against the new
    DRP endpoint 

CLEANUP:
--------

You can cleanup/reset the 5min-drp/ directory back to "factory
defaults" with the following:


  bin/terraform destroy --force
  bin/control.sh cleanup        # restores ~/.terraformrc backup
  bin/control.sh extra-cleanup


WARNING:  THIS NUKES things - including your SSH keys, which 
          means you may lose access to your nodes !!!!!!!!!!


NOTES:
------

 CONTENT  By default demo-run.sh will have the DRP endpoint
          download content from the endpoint - in many 
          situations - you'd download content to your laptop
          and push the content to the DRP endpoint (eg the
          endpoint has no direct internet connection). 

          To force "proxy" pushing content - call the run-demo.sh
          script with the "local" argument as ARGv2


