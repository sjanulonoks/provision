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


OVERVIEW AND IMPORTANT REQUIREMENTS
-----------------------------------

You will need to perform the following requirements in preparation to
using this demo 5min-drp process - details on these steps is provided
further down in this README:

  * get the 5min-drp code from the github repo

  * have a packet.net account, and your API KEY, and PROJECT ID

  * get your RackN USERNAME for registered content download authorization

  * download the 'terraform-provider-packet' plugin for terraform
    and stage it in the 'private-content' directory


WARNING
-------

Because Terraform has some ... oddities - we must write a $HOME/.terraformrc
file to be able to use the 'terraform-provider-packet' plugin. 

We will make a backup copy of your existing .terraformrc file - and we will 
restore it at the end of the demo when you run the "control.sh cleanup" 
function.  

IF YOU DO NOT run the 'cleanup' function - you will need to manually restore 
your terraform configuration file. (backup is:  $HOME/.terraform.5min-backup )


GIT CLONE
---------

The following steps will clone this content from the digitalrebar/provision 
github repo (we assume you will run this from $HOME/5min-drp - adjust yourself
accordingly if you want to put it somewhere else):

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


GET TERRAFORM-PROVIDER-PACKET PLUGIN
------------------------------------

Sadly ... you must do it this way ... 

  * you must install Go Lang as all terraform providers must be
    built from scratch:
     + install Go Lang 1.9.0 or newer according to the docs; from:
       https://golang.org/dl/
 
  * build the provider
     + build the provider:
       go get -u github.com/terraform-providers/terraform-provider-packet
     + by default this will put the compiled provider in:
         $HOME/go/bin/
     + copy the provider to the 'private-content' directory
       cp $HOME/go/bin/terraform-provider-packet private-content/



FINAL CHECK BEFORE RUNNING
--------------------------

  * Make sure you have the RackN registered user content in the 'private-contents'
    directory, as follows (file sizes/etc may vary):

      ls -l private-content/
      -rw-r--r--   1 shane  staff       781 Oct 12 18:45 secrets
      -rwxr-xr-x   1 shane  staff  22550724 Oct 12 18:48 terraform-provider-packet

  * make sure you've modified the 'secrets' file appropriately 


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
          script with the "local" argument as ARGv2.  Note that
          this options _should_ work but is not very well 
          tested (hint: there are probably some minor bugs)


ADVANCED USAGE OPTIONS
----------------------

"demo-run.sh" just drives the "bin/control.sh" script to make it easy and prettier.
You can run the full demo without any Confirmation prompts, set CONFIRM variable to
"no":

  CONFIRM=no ./demo-run.sh

The entire demo will run through without (hopefully...) any interactions. 


You can manually drive some things with the "bin/control.sh" script - simply run
it with the "--usage" or "--help" flags, it'll print out usage statement. 


"bin/control.sh cleanup" has an 8 second safety timer in it.  If you know what you're
doing - you can simply call it with "bin/control.sh cleanup force" - and it'll skip 
the safety timer. 


You can get your DRP Endpoint provioned IP address with the "bin/control.sh" script
(AFTER it has been successfully provisioned, of course):

  DRPID=`bin/control.sh get-drp-id`
  DRPIP=`bin/control.sh get-address $DRPID`


You can SSH directly to the DRP Endpoint using the injected SSH keys:

  ssh -x -i ./5min-drp-ssh-key root@$DRPIP 
  OR
  bin/control.sh ssh $DRPID



