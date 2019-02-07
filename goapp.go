package easycontainers

import (
	"bytes"
	"fmt"
	"math/rand"
	"os"
	"os/exec"

	"path"
	"time"
)

// GoApp is a containerized version of the specified Go application.
//
// AppDir is the path to the Go project, relative to the GOPATH.
//
// BuildDir is the path to the package that builds the binary, relative to the project root, not the GOPATH.
//
// Environment is a map of environment variables to create in the container.
type GoApp struct {
	ContainerName string
	Port          int
	AppDir        string
	BuildDir      string
	Environment   map[string]string
}

// NewLocalstack returns a new instance of Localstack and the port it will be using, which is
// a randomly selected number between 5000-6000.
//
// Conflicts are possible because it doesn't check if the port is already allocated.
func NewGoApp(name, appDir, buildDir string) (r *GoApp, port int) {
	port = 5000 + rand.Intn(1000)

	return &GoApp{
		ContainerName: prefix + path.Base(buildDir) + "-goapp-" + name,
		Port:          port,
		AppDir:        path.Join(GoPath(), appDir),
		BuildDir:      buildDir,
	}, port
}

// NewGoAppWithPort returns a new instance of MySQL using the specified port.
func NewGoAppWithPort(name string, port int, app, binary string) *GoApp {
	return &GoApp{
		ContainerName: prefix + path.Base(binary) + "-goapp-" + name,
		Port:          port,
		AppDir:        path.Join(GoPath(), app),
		BuildDir:      binary,
	}
}

// Container spins up the application container and runs. When the method exits, the
// container is stopped and removed.
func (g *GoApp) Container(f func() error) error {
	CleanupContainer(g.ContainerName)
	defer CleanupContainer(g.ContainerName)

	var cmdList []*exec.Cmd

	var runContainerCmd *exec.Cmd
	{
		var environmentArgs []string
		for k, v := range g.Environment {
			environmentArgs = append(environmentArgs, "-e", fmt.Sprintf("%s=%s", k, v))
		}

		runArgs := []string{
			"run",
			"--rm",
			"-p",
			fmt.Sprintf("%d:%d", g.Port, g.Port),
			"--name",
			g.ContainerName,
			"-d",
			"-it",
			"-w",
			g.AppDir,
		}

		runArgs = append(runArgs, environmentArgs...)
		runArgs = append(
			runArgs,
			"-e",
			fmt.Sprintf("GOPATH=%s", GoPath()),
			"golang:alpine",
		)

		runContainerCmd = exec.Command(
			"docker",
			runArgs...,
		)
	}

	copyProjectCmd := exec.Command(
		"/bin/bash",
		"-c",
		fmt.Sprintf(
			`docker cp %s/. $(docker ps --filter="name=^/%s$" --format="{{.ID}}"):%s`,
			g.AppDir,
			g.ContainerName,
			g.AppDir,
		),
	)

	buildProjectCmd := strCmdForContainer(
		g.ContainerName,
		fmt.Sprintf("go build %s", g.BuildDir),
	)

	cmdList = append(
		cmdList,
		runContainerCmd,
		copyProjectCmd,
		buildProjectCmd,
	)

	for _, c := range cmdList {
		err := RunCommandWithTimeout(c, 1*time.Minute)
		if err != nil {
			return err
		}
	}

	{
		runBinaryCmd := strCmdForContainer(
			g.ContainerName,
			fmt.Sprintf("./%s", path.Base(g.BuildDir)),
		)

		b := bytes.Buffer{}
		runBinaryCmd.Stderr = &b
		runBinaryCmd.Stdout = os.Stdout
		err := runBinaryCmd.Start()
		if err != nil {
			return err
		}
	}

	{
		waitForAppInitialization := strCmdForContainer(
			g.ContainerName,
			fmt.Sprintf(
				"until (curl http://localhost:%d/health/) do echo 'waiting for app to be up'; sleep 1; done",
				g.Port,
			),
		)

		err := RunCommandWithTimeout(waitForAppInitialization, 1*time.Minute)
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully created goapp container")

	return f()
}
