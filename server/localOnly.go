// +build !windows

package server

import "syscall"

func onlyICanReadThings() {
	syscall.Umask(0077)
}
