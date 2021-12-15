//go:build windows
// +build windows

package cliutils

import (
	"os/exec"
	"syscall"
)

func Setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{CreationFlags: syscall.CREATE_NEW_PROCESS_GROUP}
}
