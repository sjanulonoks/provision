#!/bin/bash

# super stupid simple color codes

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
