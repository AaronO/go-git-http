package githttp

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
)

func RunCommandMust(command string, args ...string) (string, string) {
	stdout, stderr, err := RunCommand(command, args...)

	if err != nil {
		errText := fmt.Sprintf("error: %v\nstdout: %v\nstderr: %v\n", err, stdout, stderr)
		log.Fatal(errors.New(errText))
	}
	return stdout, stderr
}

func RunCommand(command string, args ...string) (string, string, error) {
	cmd := exec.Command(command, args...)

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	return stdout.String(), stderr.String(), err
}

func RunCommandWithWd(dir, command string, args ...string) (string, string, error) {
	wd, err := os.Getwd()
	fatalError(err)

	err = os.Chdir(dir)
	fatalError(err)

	stdout, stderr, err := RunCommand(command, args...)

	err = os.Chdir(wd)
	fatalError(err)
	return stdout, stderr, err
}

func fatalError(err error) {
	if err != nil {
		log.Fatal(err)
	}
}
