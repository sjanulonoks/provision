#!/usr/bin/env bash

set -x

# Figure out what Linux distro we are running on.
export OS_TYPE= OS_VER= OS_NAME=

if [[ -f /etc/os-release ]]; then
    . /etc/os-release
    OS_TYPE=${ID,,}
    OS_VER=${VERSION_ID,,}
elif [[ -f /etc/lsb-release ]]; then
    . /etc/lsb-release
    OS_VER=${DISTRIB_RELEASE,,}
    OS_TYPE=${DISTRIB_ID,,}
elif [[ -f /etc/centos-release || -f /etc/fedora-release || -f /etc/redhat-release ]]; then
    for rel in centos-release fedora-release redhat-release; do
        [[ -f /etc/$rel ]] || continue
        OS_TYPE=${rel%%-*}
        OS_VER="$(egrep -o '[0-9.]+' "/etc/$rel")"
        break
    done

    if [[ ! $OS_TYPE ]]; then
        echo "Cannot determine Linux version we are running on!"
        exit 1
    fi
elif [[ -f /etc/debian_version ]]; then
    OS_TYPE=debian
    OS_VER=$(cat /etc/debian_version)
elif [[ $(uname -s) == Darwin ]] ; then
    OS_TYPE=darwin
    OS_VER=$(sw_vers | grep ProductVersion | awk '{ print $2 }')
fi
OS_NAME="$OS_TYPE-$OS_VER"

case $OS_TYPE in
    centos|redhat|fedora) OS_FAMILY="rhel";;
    debian|ubuntu) OS_FAMILY="debian";;
    *) OS_FAMILY=$OS_TYPE;;
esac

ensure_packages() {
    if [[ $OS_FAMILY == darwin ]] ; then
        VER=$(tar -h | grep "bsdtar " | awk '{ print $2 }' | awk -F. '{ print $1 }')
        if [[ $VER != 3 ]] ; then
            echo "Please update tar to greater than 3.0.0"
            echo "E.g: brew install libarchive"
            exit 1
        fi

        if [ "${BASH_VERSINFO}" -lt 4 ] ; then
            echo "Must have a bash version of 4 or higher"
            echo "E.g: brew install bash"
            exit 1
        fi

        if ! which 7z &>/dev/null; then
            echo "Must have 7z"
            echo "E.g: brew install p7zip"
            exit 1
        fi

    else
        if ! which bsdtar &>/dev/null; then
            if [[ $OS_FAMILY == rhel ]] ; then
                sudo yum install -y bsdtar
            elif [[ $OS_FAMILY == debian ]] ; then
                sudo apt-get install -y bsdtar
            fi
        fi
        if ! which 7z &>/dev/null; then
            if [[ $OS_FAMILY == rhel ]] ; then
                sudo yum install -y p7zip
            elif [[ $OS_FAMILY == debian ]] ; then
                sudo apt-get install -y p7zip
            fi
        fi
    fi
}

case $(uname -s) in
    Darwin)
        binpath="bin/darwin/amd64"
        bindest="/usr/local/bin"
        # Someday, handle adding all the launchd stuff we will need.
        shasum="command shasum -a 256";;
    Linux)
        binpath="bin/linux/amd64"
        bindest="/usr/local/bin"
        if [[ -d /etc/systemd/system ]]; then
            # SystemD
            initfile="startup/dr-provision.service"
            initdest="/etc/systemd/system/dr-provision.service"
            starter="sudo systemctl daemon-reload && sudo systemctl start dr-provision"
            enabler="sudo systemctl daemon-reload && sudo systemctl enable dr-provision"
        elif [[ -d /etc/init ]]; then
            # Upstart
            initfile="startup/dr-provision.unit"
            initdest="/etc/init/dr-provision.conf"
            starter="sudo service dr-provision start"
            starter="sudo service dr-provision enable"
        elif [[ -d /etc/init.d ]]; then
            # SysV
            initfile="startup/dr-provision.sysv"
            initdest="/etc/init.d/dr-provision"
            starter="/etc/init.d/dr-provision start"
            starter="/etc/init.d/dr-provision enable"
        else
            echo "No idea how to install startup stuff -- not using systemd, upstart, or sysv init"
            exit 1
        fi
        shasum="command sha256sum";;
    *)
        # Someday, support installing on Windows.  Service creation could be tricky.
        echo "No idea how to check sha256sums";;
esac

case $1 in
     install)
             ensure_packages
             $shasum -c sha256sums || exit 1
             sudo cp "$binpath"/* "$bindest"
             if [[ $initfile ]]; then
                 sudo cp "$initfile" "$initdest"
                 echo "You can start the DigitalRebar Provision service with:"
                 echo "$starter"
                 echo "You can enable the DigitalRebar Provision service with:"
                 echo "$enabler"
             fi;;
     remove)
         sudo rm -f "$bindest/dr-provision" "$bindest/drpcli" "$initdest";;
     *)
         echo "Unknown action \"$1\". Please use 'install' or 'remove'";;
esac

