#!/usr/bin/env bash

set -e

DEFAULT_RS_VERSION="tip"

usage() {
        echo
	echo "Usage: $0 [--rs-version=<Version to install>] [--isolated] <install|remove>"
        echo
        echo "Options:"
        echo "  --debug=[true|false] # Enables debug output"
        echo "  --isolated # Sets up the current directory as a place to the cli and provision"
        echo "  --rs-version=<string>  # Version identifier if downloading.  Defaults to $DEFAULT_RS_VERSION"
        echo
        echo "  install    # Sets up an insolated or system enabled install.  Outputs nexts steps"
        echo "  remove     # Removes the system enabled install.  Requires no other flags"
	echo "Defaults are: "
	echo "  version = tip (instead of v2.9.1003)"
        echo "  isolated = false"
        echo "  force = false"
        echo "  debug = false"
	exit 1
}

RS_VERSION=$DEFAULT_RS_VERSION
ISOLATED=false
args=()
while (( $# > 0 )); do
    arg="$1"
    arg_key="${arg%%=*}"
    arg_data="${arg#*=}"
    case $arg_key in
        --help|-h)
            usage
            exit 0
            ;;
        --isolated)
            ISOLATED=true
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

ensure_packages() {
    echo "Ensuring required tools are installed"
    if [[ $OS_FAMILY == darwin ]] ; then
        VER=$(tar -h | grep "bsdtar " | awk '{ print $2 }' | awk -F. '{ print $1 }')
        if [[ $VER != 3 ]] ; then
            echo "Please update tar to greater than 3.0.0"
            echo 
            echo "E.g: "
            echo "  brew install libarchive --force"
            echo "  brew link libarchive --force"
            echo
            echo "Close current terminal and open a new terminal"
            echo
            exit 1
        fi
        if ! which 7z &>/dev/null; then
            echo "Must have 7z"
            echo "E.g: brew install p7zip"
            exit 1
        fi
    else
        if ! which bsdtar &>/dev/null; then
            echo "Installing bsdtar"
            if [[ $OS_FAMILY == rhel ]] ; then
                sudo yum install -y bsdtar
            elif [[ $OS_FAMILY == debian ]] ; then
                sudo apt-get install -y bsdtar
            fi
        fi
        if ! which 7z &>/dev/null; then
            echo "Installing bsdtar"
            if [[ $OS_FAMILY == rhel ]] ; then
                sudo yum install -y p7zip
            elif [[ $OS_FAMILY == debian ]] ; then
                sudo apt-get install -y p7zip-full
            fi
        fi
    fi
}

case $(uname -s) in
    Darwin)
        binpath="bin/darwin/amd64"
        bindest="/usr/local/bin"
        tar="command bsdtar"
        # Someday, handle adding all the launchd stuff we will need.
        shasum="command shasum -a 256";;
    Linux)
        binpath="bin/linux/amd64"
        bindest="/usr/local/bin"
        tar="command bsdtar"
        if [[ -d /etc/systemd/system ]]; then
            # SystemD
            initfile="assets/startup/dr-provision.service"
            initdest="/etc/systemd/system/dr-provision.service"
            starter="sudo systemctl daemon-reload && sudo systemctl start dr-provision"
            enabler="sudo systemctl daemon-reload && sudo systemctl enable dr-provision"
        elif [[ -d /etc/init ]]; then
            # Upstart
            initfile="assets/startup/dr-provision.unit"
            initdest="/etc/init/dr-provision.conf"
            starter="sudo service dr-provision start"
            enabler="sudo service dr-provision enable"
        elif [[ -d /etc/init.d ]]; then
            # SysV
            initfile="assets/startup/dr-provision.sysv"
            initdest="/etc/init.d/dr-provision"
            starter="/etc/init.d/dr-provision start"
            enabler="/etc/init.d/dr-provision enable"
        else
            echo "No idea how to install startup stuff -- not using systemd, upstart, or sysv init"
            exit 1
        fi
        shasum="command sha256sum";;
    *)
        # Someday, support installing on Windows.  Service creation could be tricky.
        echo "No idea how to check sha256sums"
        exit 1;;
esac

case $1 in
     install)
             ensure_packages
             # Are we in a build tree
             if [ -e server ] ; then
                 if [ ! -e bin/linux/amd64/drpcli ] ; then
                     echo "It appears that nothing has been built."
                     echo "Please run tools/build.sh and then rerun this command".
                     exit 1
                 fi
             else
                 # We aren't a build tree, but are we extracted install yet?
                 # If not, get the requested version.
                 if [[ ! -e sha256sums || $force ]] ; then
                     echo "Installing Version $RS_VERSION of Digital Rebar Provision"
                     curl -sfL -o dr-provision.zip https://github.com/digitalrebar/provision/releases/download/$RS_VERSION/dr-provision.zip
                     curl -sfL -o dr-provision.sha256 https://github.com/digitalrebar/provision/releases/download/$RS_VERSION/dr-provision.sha256

                     $shasum -c dr-provision.sha256
                     $tar -xf dr-provision.zip
                 fi
                 $shasum -c sha256sums || exit 1
             fi

             if [[ $ISOLATED == false ]] ; then
                 sudo cp "$binpath"/* "$bindest"
                 if [[ $initfile ]]; then
                     sudo cp "$initfile" "$initdest"
                     echo "You can start the DigitalRebar Provision service with:"
                     echo "$starter"
                     echo "You can enable the DigitalRebar Provision service with:"
                     echo "$enabler"
                 fi
             else
                 mkdir -p drp-data

                 # Make local links for execs
                 rm -f drpcli dr-provision
                 ln -s $binpath/drpcli drpcli
                 ln -s $binpath/dr-provision dr-provision

                 if [[ $IPADDR == "" ]] ; then
                     if [[ $OS_FAMILY == darwin ]]; then
                         echo "On Darwin, must specify --static-ip"
                     else
                         gwdev=$(/sbin/ip -o -4 route show default |head -1 |awk '{print $5}')
                         if [[ $gwdev ]]; then
                             # First, advertise the address of the device with the default gateway
                             IPADDR=$(/sbin/ip -o -4 addr show scope global dev "$gwdev" |head -1 |awk '{print $4}')
                         else
                             # Hmmm... we have no access to the Internet.  Pick an address with
                             # global scope and hope for the best.
                             IPADDR=$(/sbin/ip -o -4 addr show scope global |head -1 |awk '{print $4}')
                         fi

                         IPADDR="--static-ip=${IPADDR///*}"
                     fi
                 fi

                 echo "Run the following commands to start up dr-provision in a local isolated way."
                 echo "The server will store information and server files in the drp-data directory."
                 echo
                 echo "sudo ./dr-provision $IPADDR --file-root=`pwd`/drp-data/tftpboot --data-root=drp-data/digitalrebar &"
                 echo "tools/discovery-load.sh"
             fi;;
     remove)
         sudo rm -f "$bindest/dr-provision" "$bindest/drpcli" "$initdest";;
     *)
         echo "Unknown action \"$1\". Please use 'install' or 'remove'";;
esac

