//go:build !windows
// +build !windows

package cliutils

import (
	"os/exec"
	"syscall"
)

func Setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
