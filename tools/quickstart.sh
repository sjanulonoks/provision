#!/usr/bin/env bash

set -e

usage() {
	echo "Usage: $0 [--version=<Version to install>] [--static-ip=<published ip>]"
	echo "Defaults are: "
	echo "  version = tip (instead of v2.9.1003)"
	echo "  static-ip = IP of interface with the default gateway or first global address"
	exit 1
}

IPADDR=""
VERSION="tip"
args=()
while (( $# > 0 )); do
    arg="$1"
    arg_key="${arg%%=*}"
    arg_data="${arg#*=}"
    case $arg_key in
        --static-ip)
            IPADDR=$arg
            ;;
        --help|-h)
            usage
            exit 0
            ;;
        --*)
            arg_key="${arg_key#--}"
            arg_key="${arg_key//-/_}"
            arg_key="${arg_key^^}"
            echo "Overriding $arg_key with $arg_data"
            export $arg_key="$arg_data"
            ;;
        *)
            args+=("$arg");;
    esac
    shift
done
set -- "${args[@]}"

if [[ $DEBUG == true ]] ; then
    set -x
fi

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

if [[ $IPADDR == "" ]] ; then
    if [[ $OS_FAMILY == darwin ]]; then
            echo "On Darwin, must specify --static-ip"
            usage
    fi
    gwdev=$(/sbin/ip -o -4 route show default |head -1 |awk '{print $5}')
    if [[ $gwdev ]]; then
        # First, advertise the address of the device with the default gateway
	IPADDR=$(/sbin/ip -o -4 addr show scope global dev "$gwdev" |head -1 |awk '{print $4}')
    else
        # Hmmm... we have no access to the Internet.  Pick an address with
        # global scope and hope for the best.
	IPADDR=$(/sbin/ip -o -4 addr show scope global |head -1 |awk '{print $4}')
    fi

    IPADDR="--static-ip=$IPADDR"
fi


ensure_packages() {
    # On Macs, tar is bsdtar, but we need a good enough version
    if [[ $OS_FAMILY == darwin ]] ; then
        VER=$(tar -h | grep "bsdtar " | awk '{ print $2 }' | awk -F. '{ print $1 }')
        if [[ $VER != 3 ]] ; then
            echo "Please update tar to greater than 3.0.0"
            echo "E.g: brew install libarchive"
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
    fi

    if ! which curl &>/dev/null; then
        echo "Installing curl ..."
        if [[ $OS_FAMILY == rhel ]] ; then
            sudo yum install -y curl 2>/dev/null >/dev/null
        elif [[ $OS_FAMILY == debian ]] ; then
            sudo apt-get install -y curl 2>/dev/null >/dev/null
            sudo updatedb 2>/dev/null >/dev/null
        fi

        if ! which curl &>/dev/null; then
            echo "Please install curl!"
            if [[ $(uname -s) == Darwin ]] ; then
                echo "Something like: brew install curl"
            fi
            exit 1
        fi
    fi
}

case $(uname -s) in
    Darwin)
        binpath="bin/darwin/amd64"
        tar="command tar"
        shasum="command shasum -a 256";;
    Linux)
        binpath="bin/linux/amd64"
        tar="command bsdtar"
        shasum="command sha256sum";;
    *)
        # Someday, support installing on Windows.  Service creation could be tricky.
        echo "No idea how to check sha256sums"
        exit 1;;
esac

ensure_packages
curl -sfL -o dr-provision.zip https://github.com/digitalrebar/provision/releases/download/$VERSION/dr-provision.zip
curl -sfL -o dr-provision.sha256 https://github.com/digitalrebar/provision/releases/download/$VERSION/dr-provision.sha256

$shasum -c dr-provision.sha256
$tar -xf dr-provision.zip
$shasum -c sha256sums

rm -f drpcli dr-provision
ln -s $binpath/drpcli drpcli
ln -s $binpath/dr-provision dr-provision

mkdir -p drp-data

echo "Run the following commands to start up dr-provision in a local isolated way."
echo "The server will store information and server files from the drp-data directory."
echo
echo "sudo ./dr-provision $IPADDR --file-root=`pwd`/drp-data/tftpboot --data-root=drp-data/digitalrebar &"
echo "./discovery-load.sh"

