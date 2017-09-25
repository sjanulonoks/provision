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
#  installs a DRP server  
###

PATH=`pwd`/bin:$PATH

VER_DRP=${VER_DRP:-"stable"}

os_type=`uname -s`
os_arch=`uname -i`
[[ $os_arch == "x86_64" ]] && os_arch="amd64"
os_type=${os_type,,}
os_arch=${os_arch,,}

set -x
curl -fsSL https://raw.githubusercontent.com/digitalrebar/provision/${VER_DRP}/tools/install.sh \
  | bash -s -- --isolated install --drp-version=${VER_DRP}
set +x

ln -s `pwd`/bin/${os_type}/${os_arch}/drpcli bin/drpcli
ln -s `pwd`/bin/${os_type}/${os_arch}/dr-provision bin/dr-provision

cat <<EOFSTART > drp-start.sh
#!/bin/bash

PATH=`pwd`/bin:$PATH
set -x
dr-provision --base-root=$HOME/drp-data --local-content="" --default-content="" > $HOME/drp-local.log 2>&1 &
set +x
sleep 2
echo "Process listing for 'dr-provision':"
ps -ef | grep -v grep | grep --color=always dr-provision

EOFSTART

chmod 755 drp-start.sh
./drp-start.sh

cat <<'EOFISOS' > drp-isos.sh
#!/bin/bash

# for 5min demo we don't want any community content - we use RackN 
# content for pacekt.net functionality

#ISOS="ce-ubuntu-16.04-install ce-centos-7.3.1611-install ce-sledgehammer"
#ISOS="ce-centos-7.3.1611-install ce-sledgehammer"

if [[ -n "$ISOS" ]]
then
  echo "Downloading and installing bootenvs ISOS now ... please be patient ... "
  for ISO in $ISOS
  do
    set -x
    ./drpcli bootenvs uploadiso $ISO
    set +x
  done
else
  echo "No ISOS specified .... skipping ISO upload step ... "
fi

exit 0
EOFISOS

chmod 755 drp-isos.sh
./drp-isos.sh

./drpcli isos list

exit $?
