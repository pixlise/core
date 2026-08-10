package jobrunner

import (
	"os/exec"
)

func runCommand(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	cmdStdOut, err := cmd.CombinedOutput()
	return string(cmdStdOut), err
}
