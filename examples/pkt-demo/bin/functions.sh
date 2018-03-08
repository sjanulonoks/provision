#!/bin/bash

###
#  various useful helper functions
###

###
#  exit with error code and FATAL message
#  example:  xiterr 1 "error message"
###
function xiterr() { [[ $1 =~ ^[0-9]+$ ]] && { local _xit=$1; shift; } || local _xit=255; echo "FATAL: $*"; exit $_xit; }

###
#  set global XIT value if we receive an error exit code (anything 
#  besides a Zero)
###
function xit() { local _x=$1; (( $_x )) && (( XIT += _x )); }

# set our global _OS_* variables for re-use
function os_info() {
	# Figure out what Linux distro we are running on.
	# set these globally for use outside of the script
	export _OS_TYPE= _OS_VER= _OS_NAME= _OS_FAMILY=

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

###
#  simple helper loop functions, expects:
#
#   UUIDS = global var with space separated list of UUIDS to operate on
#   $1    = `machines` sub-command to run
#   $2    = action/argument to sub-commnand
#
#  example:
#
#     export UUIDS="uuid_1 uuid_2 ... uuid_N"
#     my_machines addprofile krib-cluster-live
#
#  which runs:
#
#     drpcli machines addprofile <UUID> krib-cluster-live
#
#  $2 can be empty if it doesnot apply (eg 'my_machines show')
###
function my_machines() { local _u; for _u in $UUIDS; do set -x; drpcli machines $1 $_u $2; set +x; done; }

###
#  simple function to request a new CLUSTER_NAME be set if the
#  name is set to the defaul of "demo".  No additional checking
#  will be done, so operator can override, and keep "demo" if
#  desired.  
#
#  This function expects ARGv1 to be set to the value of CLUSTER_NAME
#  and if that value is "demo", prompt for a change.
###
function set_cluster_name() {
  local _cn

  case $1 in
    [dD][eE][mM][oO])
      echo ""
      echo "The CLUSTER NAME is set to the default value of 'demo'."
      echo "You are STRONGLY urged to select a very short name that"
      echo "is uniuque to your cluster - otherwise, multiple clusters"
      echo "with the same name will result in Nodes built with the"
      echo "same name.  This is not fatal, but it can cause confusion"
      echo "when you view Nodes in the Packet portal"
      echo ""
      echo "We recommend keeping it to LESS than 8 to 12 characters or"
      echo "less; with ONLY a dash ('-') as a special character."
      echo ""
      echo "examples: pkt-demo"
      echo "          cluster01"
      echo ""
      read -p "Set a new cluster name: " _cn
      echo ""

      if [[ "${_cn}" != "${1}" ]]
      then
        # fixup vars.tf with new cluster name
        sed -i "" "s/^\(variable \"cluster_name\" { default = \"\).*\(\" }\)$/\1${_cn}\2/g" vars.tf
      fi

      export CLUSTER_NAME="${_cn}"
      ;;
  esac
} # end set_cluster_name()

###
#  accept as ARGv1 a sha256 check sum file to test - files will be tested
#  relative to the path/filename found in the SHA256 sum file
###
function check_sum() {
  local _sum=$1
  [[ -z "$_sum" ]] && xiterr 1 "no check sum file passed to check_sum()"
  [[ ! -r "$_sum" ]] && xiterr 1 "unable to read check sum file '$_sum'"
  local _platform=`uname -s`

  case $_platform in
    Darwin) shasum -a 256 -c $_sum; xit $? ;;
    Linux)  sha256sum -c $_sum; xit $? ;;
    *) xiterr 2 "unsupported platform type '$_platform'"
  esac

  (( $XIT )) && xiterr 9 "checksum failed for '$_sum'"
}

###
#  Compares two tuple-based, dot-delimited version numbers a and b (possibly
#  with arbitrary string suffixes). 
#
#  Returns:
#    1  if a < b
#    2  if equal
#    3  if a > b
#
#  Everything after the first character not in [0-9.] is compared
#  lexicographically using ASCII ordering if the tuple-based versions are equal.
###
function _compare_versions() {
    if [[ $1 == $2 ]]; then
        return 2
    fi
    local IFS=.
    local i a=(${1%%[^0-9.]*}) b=(${2%%[^0-9.]*})
    local arem=${1#${1%%[^0-9.]*}} brem=${2#${2%%[^0-9.]*}}
    for ((i=0; i<${#a[@]} || i<${#b[@]}; i++)); do
        if ((10#${a[i]:-0} < 10#${b[i]:-0})); then
            return 1
        elif ((10#${a[i]:-0} > 10#${b[i]:-0})); then
            return 3
        fi
    done
    if [ "$arem" '<' "$brem" ]; then
        return 1
    elif [ "$arem" '>' "$brem" ]; then
        return 3
    fi
    return 2
}

###
#   usage:  [ -s ] VALUE_1 'comparison' VALUE_2
#           -s    [optional] - do not echo strings 'true' or 'false'
#
#   for given input, return:
#
#     input                return
#     ---------------------------
#     0.10.0 '=' 0.10.0    true
#     0.10.1 '=' 0.10.0    false
#     0.10.7 '>' 0.10.5    true
#     0.10.7 '<' 0.10.5    false
#
#  Return code set to: '0' for 'true'
#                      '1' for 'fasle'
###
function compver() {
  local _op _val _true _false
  case $1 in
    -s|-silent|--silent|-q|-quiet|--quiet|silent|quiet)
      shift 1
      _true=""
      _false=""
      ;;
    *)
      _true="echo true"
      _false="echo false"
      ;;
  esac

  _compare_versions $1 $3
  _val=$?
  case $_val in
    1) _op='<';;
    2) _op='=';;
    3) _op='>';;
  esac
  [[ "$2" == "$_op" ]] && { $_true; return 0; } || { $_false; return 1; }
}


###
#  super stupid simple color codes
#
#  example:  cprintf $green "some green text\n"
#
#    notes:  like printf(), you must add trailing newline
###
black='\E[30;40m'
red='\E[31;40m'
green='\E[32;40m'
yellow='\E[33;40m'
blue='\E[34;40m'
magenta='\E[35;40m'
cyan='\E[36;40m'
white='\E[37;40m'
bold='\E[37;1m'
underline='\E[37;4m'

#  Reset text attributes to normal
function _reset() { tput sgr0; }
#function _reset() { echo -ne \E[0m; }

# $1 = color; $2 = message
function cprintf () {
	local _default_msg="No message passed."
	_color=${1:-$black}
	shift 
	_message=${*:-$_default_msg}

	printf "${_color}${_message}$(_reset)"

	return
} 


###
#  DO NOT DELETE THIS FUNCTION ... it is used as a helper to 
#  determine if we've been sourced once before, and avoicd being
#  sourced again ... 
###
function functions() {
  local _i_exist
}
