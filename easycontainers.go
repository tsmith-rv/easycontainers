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
	"os/signal"
	"syscall"
)

const prefix = "easycontainers-"

func init() {
	rand.Seed(time.Now().Unix())

	CleanupAllContainers()

	signalCh := make(chan os.Signal, 1024)
	signal.Notify(signalCh, syscall.SIGHUP, syscall.SIGUSR2, syscall.SIGINT, syscall.SIGQUIT, syscall.SIGKILL)

	go func() {
		<-signalCh

		CleanupAllContainers()
	}()
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

// CleanupAllContainers will stop all containers starting with prefix
func CleanupAllContainers() error {
	cmd := exec.Command(
		"docker",
		"stop",
		prefix,
	)

	var b bytes.Buffer
	cmd.Stderr = &b

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error in command : %s -- %s", err, b.String())
	}

	return err
}

// CleanupContainer stops the container with the specified name.
func CleanupContainer(name string) error {
	cmd := exec.Command(
		"/bin/bash",
		"-c",
		fmt.Sprintf(`docker stop $(docker ps --filter="name=^/%s$" --format="{{.ID}}")`, name),
	)

	var b bytes.Buffer
	cmd.Stderr = &b

	err := cmd.Run()
	if err != nil {
		return fmt.Errorf("error in command : %s -- %s", err, b.String())
	}

	return err
}

// Logs runs the docker logs command on the specified container and returns the output
func Logs(name string) string {
	cmd := exec.Command(
		"docker",
		"logs",
		name,
	)

	var outputBuf bytes.Buffer
	cmd.Stderr = &outputBuf
	cmd.Stdout = &outputBuf

	cmd.Run()

	return outputBuf.String()
}

// RunCommandWithTimeout will execute the specified cmd, but will timeout and
// return and error after 1 minute.
func RunCommandWithTimeout(cmd *exec.Cmd, timeout time.Duration) error {
	finish := make(chan error)
	timer := time.NewTimer(timeout)

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
	case <-timer.C:
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
