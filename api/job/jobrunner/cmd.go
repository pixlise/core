package jobrunner

import (
	"os/exec"
)

type CommandRunner func(command string, args []string) (string, error)

func runCommand(command string, args []string) (string, error) {
	cmd := exec.Command(command, args...)
	cmdStdOut, err := cmd.CombinedOutput()
	return string(cmdStdOut), err
}
