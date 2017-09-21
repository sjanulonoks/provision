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

set -x 
curl -fsSL https://raw.githubusercontent.com/digitalrebar/provision/tip/tools/install.sh \
  | bash -s -- --isolated install --drp-version=tip

#ln -s ~/bin/linux/amd64/drpcli bin/drpcli
#ln -s ~/bin/linux/amd64/dr-provision bin/dr-provision

cat <<EOFSTART > drp-start.sh
#!/bin/bash

set -x
sudo ./dr-provision --base-root=$HOME/drp-data --local-content="" --default-content="" > $HOME/drp-local.log 2>&1 &
set +x
sleep 2
ps -ef | grep -v grep | grep --color=always dr-provision

EOFSTART

chmod 755 drp-start.sh
./drp-start.sh

cat <<'EOFISOS' > drp-isos.sh
#!/bin/bash

BASE=https://raw.githubusercontent.com/digitalrebar/provision-content/master/bootenvs

echo "Installing bootenvs now ... please be patient ... "
for BOOTENV in ce-centos-7.3.1611.yml ce-discovery.yml ce-sledgehammer.yml ce-ubuntu-16.04.yml
do
        curl -s $BASE/$BOOTENV -o $BOOTENV
        NAME=`cat $BOOTENV | grep "^Name: " | cut -d '"' -f 2`
set -x
        ./drpcli bootenvs create - < $BOOTENV
        ./drpcli bootenvs uploadiso $NAME
set +x
done

exit 0
EOFISOS

chmod 755 drp-isos.sh
./drp-isos.sh

./drpcli isos list

