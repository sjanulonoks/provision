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


README file for pkt-demo

NOTE:  This example used to be called "5min-demo".  It has been renamed
       to "pkt-demo" to more closely align with the use case, and 
       expanded scope.


OVERVIEW AND IMPORTANT REQUIREMENTS
-----------------------------------

You will need to perform the following requirements in preparation to
using this demo pkt-demo process - details on these steps is provided
further down in this README:

  * get the pkt-demo code from the github repo

  * get your API and Username secrets - modify the "secrets" file
    * have a packet.net account, and your API KEY, and PROJECT ID
    * get your RackN USERNAME for registered content download authorization

  * [optional] make changes to the terraform "vars.tf" parameters


NOTE: The terraform-provider-plugin has been updated in the public repo,
      it's no longer necessary to get/build it manually and pre-stage it.


GIT CLONE
---------

The following steps will clone this content from the digitalrebar/provision 
github repo (we assume you will run this from $HOME/pkt-demo - adjust yourself
accordingly if you want to put it somewhere else):

    git clone -n https://github.com/digitalrebar/provision.git --depth=1
    cd provision
    git checkout HEAD examples/pkt-demo
    cd examples/pkt-demo
    mv pkt-demo $HOME/
    cd ../..
    rm -rf provision
    cd $HOME/pkt-demo

SECRETS INFORMATION
-------------------

EDIT THE SECRETS FILE !!  Located in private-content/secrets.  You need:

  API       packet.net key for access to your Packet Project
  PROJECT   packet.net project to create DRP and Nodes in 
  USERNAME  your RackN username ID 

    # modify the API, PROJECT, and USERNAME  variables

    vim private-content/secrets

  API and PROJECT are from packet.net and you should find them in your
  Packet portal management

  USERNAME is from the RackN Portal - to find your USERNAME, log in 
  to the portal, and navigate to:

    Hamburger Menu (3 horizontal lines in upper left)
    User Profile
    Unique User Identifier

  Direct URL:  https://rackn.github.io/provision-ux/#/user/ 

  It will be a big ugly UUID like string like:  ad9914b7-60bd-49d9-81d0-95e532e7ce1c


  NOTE: Please do not modify the following in the 'secrets' file:
        API_KEY, PROJECT_ID, and RACKN_USERNAME 


FINAL CHECK BEFORE RUNNING
--------------------------

  * make sure you've modified the 'secrets' file appropriately 
    (inject API, PROJECT, and USERNAME)

  * make sure you've modified the 'vars.tf' file to suit your use
    (change 'cluster_name', 'machines_count', etc....)


[optional] MODIFY THE TERRAFORM "vars.tf" FILE
----------------------------------------------

  * you may optionally make changes to the "vars.tf" file - specifically, you can 
    set the "cluster_name" to something other than "demo" - if you instantiate
    multiple DRP/Machines clusters, then the names will collide in the packet.net
    portal.  Changing the "cluster_name" will help in identifying which resources
    belong to which cluster. 

  * you can modify which Operating System the DRP endpoint is running on - the only
    two supported/tested are Centos 7 and Ubuntu 16.04

  * specify the number of Machines to provision 

  * change the packet.net facility to provsion the cluster in 


RUN DEMO-RUN.SH SCRIPT
----------------------

The 'demo-run.sh' is the control script that will walk you through the deployment
process.  Simply start this script. 

    ./demo-run.sh

If you re-run the script and want to skip steps that have run previously, simply 
answer "no" to the "ACTION" input. 

USAGE options for "demo-run.sh"

  CONFIRM=no ./demo-run.sh      # disable prompting for each step - auto run
  SKIP_LOCAL=yes ./demo-run.sh  # skip installing DRP locally - if you have a
                                # current copy installed already - mostly used
                                # in bandwidth constrained environments to 
                                # avoid downloading the dr-provision.zip

  CONFIRM and SKIP_LOCAL can be combined if you choose

WHAT HAPPENS?
-------------

1.  set PATH to include the ./bin directory for DRP and terraform/etc.
2.  install terraform locally in your ./bin directory 
    (can be skipped, eg for demo in bandwidth constrained environments)
3.  install the secrets to the terraform vars.tf file
4.  create SSH keys for DRP endpoint and nodes
5.  inject the SSH keys in to packet.net Project
6.  build the DRP Endpoint server in packet.net 
    (on centos-7, configurable, see vars.tf)
7.  install DRP locally in ./bin for 'drpcli' control
8.  install DRP on the remote endpoint 
9.  configure content and services on DRP  [SEE NOTE: CONTENT]
10. set the DRP endpoint IP address to terraform 'drp-machines.tf'
11. kick over "N" number of machines to provision against the new
    DRP endpoint 
    (set "N" in vars.tf for "machines_count" variable)


CLEANUP:
--------

You can cleanup/reset the pkt-demo/ directory back to "factory
defaults" with the following:

  bin/terraform destroy --force # wipe the terraform resources
  bin/control.sh cleanup        # restores ~/.terraformrc backup


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
          script with the "local" argument as ARGv2.  Note that
          this options _should_ work but is not very well 
          tested (hint: there are probably some minor bugs)


ADVANCED USAGE OPTIONS
----------------------

"demo-run.sh" just drives the "bin/control.sh" script to make it easy 
and prettier.  You can run the full demo without any Confirmation prompts, 
set CONFIRM variable to "no":

  CONFIRM=no ./demo-run.sh

The entire demo will run through without (hopefully...) any interactions. 


If you are running in a bandwidth constrained environment (ge poor WiFi, 
or Cellular based Hotspot), and IF you have current version of DRP 
installed on your laptop (and in your PATH), then you can skip the local
download/install step, with:

  SKIP_LOCAL=yes ./demo-run.sh


CONFIRM and SKIP_LOCAL can be combined if you want.


You can manually drive some things with the "bin/control.sh" script - simply
run it with the "--usage" or "--help" flags, it'll print out usage statement. 


"bin/control.sh cleanup" has an 8 second safety timer in it.  If you know 
what you're doing - you can simply call it with "bin/control.sh cleanup 
force" - and it'll skip the safety timer. 


You can get your DRP Endpoint provisioned IP address with the "bin/control.sh" 
script (AFTER it has been successfully provisioned, of course):

  DRP_ID=`bin/control.sh get-drp-id`
  DRP_ADDR=`bin/control.sh get-address $DRP_ID`


You can SSH directly to the DRP Endpoint using the injected SSH keys:

  ssh -x -i ./drp-ssh-key root@$DRP_IP 
  OR
  bin/control.sh ssh $DRP_ID


You should be able to SSH to the Machines directly as well, using the following
(after they've been provsioned and installed, of course):

  terraform plan    # to get the various machines IP addresses
                    # or get from packet.net

  ssh -x -i ./machines-ssh-key root@<MACHINE_IP>
                  

IF you need a custom version of the terraform-provider-packet plugin, you can 
specify it in your ~/.terraformrc file like so:

  providers { packet = "$HOME/.terraform.d/terraform-provider-packet" }

Prior to running "terraform init" (which occurs during the "terraform-install"
stage in this Demo.
