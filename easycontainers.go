package easycontainers

import (
	"bytes"
	"errors"
	"math/rand"
	"os/exec"
	"strings"
	"time"

	"fmt"
	"os"

	"go/build"
)

func init() {
	rand.Seed(time.Now().Unix())
}

// GoPath returns the value stored in the GOPATH environment variable.
// If that value isn't set in the environment, it will return the value
// of build.Default.GOPATH.
func GoPath() string {
	s, exists := os.LookupEnv("GOPATH")
	if !exists {
		s = build.Default.GOPATH
	}

	return s
}

// CleanupContainer stops the container with the specified name.
//
// This will also rm the container because the --rm switch should be set on all containers.
func CleanupContainer(name string) error {
	cmd := exec.Command(
		"docker",
		"stop",
		name,
	)

	var b bytes.Buffer
	cmd.Stderr = &b

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error in command : %s -- %s", err, b.String())
	}

	return err
}

// RunCommandWithTimeout will execute the specified cmd, but will timeout and
// return and error after 1 minute.
func RunCommandWithTimeout(cmd *exec.Cmd) error {
	var (
		timeout = time.NewTimer(1 * time.Minute)
		finish  = make(chan error)
	)

	go func() {
		var err error

		defer func() {
			finish <- err
		}()

		var b bytes.Buffer
		cmd.Stderr = &b
		cmd.Stdout = os.Stdout

		err = cmd.Run()
		if err != nil {
			err = fmt.Errorf("error in command : %s -- %s", err, b.String())
			return
		}
	}()

	select {
	case err := <-finish:
		if err != nil {
			return err
		}
	case <-timeout.C:
		return errors.New("container timed out")
	}

	return nil
}

func cmdForContainer(name string, cmd *exec.Cmd) *exec.Cmd {
	return exec.Command(
		"docker",
		"exec",
		name,
		"/bin/bash",
		"-c",
		strings.Join(cmd.Args, " "),
	)
}

func strCmdForContainer(name string, str string) *exec.Cmd {
	return exec.Command(
		"docker",
		"exec",
		name,
		"/bin/bash",
		"-c",
		str,
	)
}
