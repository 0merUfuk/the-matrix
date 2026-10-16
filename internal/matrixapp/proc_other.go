//go:build !unix

package matrixapp

import "os/exec"

func ownGroup(*exec.Cmd) {}

func terminateGroup(cmd *exec.Cmd, _ bool) {
	if cmd.Process != nil {
		_ = cmd.Process.Kill()
	}
}
