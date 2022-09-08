//go:build !windows
// +build !windows

package specific

import (
	"os/exec"
	"syscall"
)

func Setpgid(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
}
