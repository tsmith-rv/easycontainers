package easycontainers

import (
	"bytes"
	"context"
	"fmt"
	"time"

	"path"

	"strconv"

	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// GoApp is a containerized version of the specified Go application.
//
// AppDir is the path to the Go project, relative to the GOPATH.
//
// BuildDir is the path to the package that builds the binary, relative to the project root, not the GOPATH.
//
// Environment is a map of environment variables to create in the container.
type GoApp struct {
	Client         *client.Client
	ContainerName  string
	Port           int
	AppDir         string
	BuildDir       string
	HealthEndpoint string
	Environment    map[string]string
}

// NewGoApp returns a new instance of GoApp and the port it will be using.
func NewGoApp(name, appDir, buildDir, healthEndpoint string) (r *GoApp, port int) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &GoApp{
		Client:         c,
		ContainerName:  prefix + path.Base(buildDir) + "-goapp-" + name,
		Port:           port,
		AppDir:         path.Join(GoPath(), appDir),
		BuildDir:       buildDir,
		HealthEndpoint: healthEndpoint,
	}, port
}

// NewGoAppWithPort returns a new instance of GoApp using the specified port.
func NewGoAppWithPort(name string, port int, app, binary, healthEndpoint string) *GoApp {
	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &GoApp{
		Client:         c,
		ContainerName:  prefix + path.Base(binary) + "-goapp-" + name,
		Port:           port,
		AppDir:         path.Join(GoPath(), app),
		BuildDir:       binary,
		HealthEndpoint: healthEndpoint,
	}
}

// Container spins up the application container and runs. When the method exits, the
// container is stopped and removed.
func (g *GoApp) Container(f func() error) error {
	ctx := context.Background()
	reader, err := g.Client.ImagePull(ctx, "docker.io/library/golang:alpine", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	var env []string
	for k, v := range g.Environment {
		env = append(env, fmt.Sprintf("%s=%s", k, v))
	}

	stopHealthCheck := make(chan struct{})

	healthCommand := path.Join(
		fmt.Sprintf("curl http://localhost:%d/", g.Port),
		g.HealthEndpoint,
	)

	resp, err := g.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: "golang:alpine",
			Env:   append(env, fmt.Sprintf("GOPATH=%s", GoPath())),
			Tty:   true,
			ExposedPorts: nat.PortSet{
				nat.Port(fmt.Sprintf("%d/tcp", g.Port)): struct{}{},
			},
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD-SHELL", healthCommand},
				Interval: 5 * time.Second,
				Timeout:  1 * time.Minute,
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				nat.Port(fmt.Sprintf("%d/tcp", g.Port)): []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: strconv.Itoa(g.Port),
					},
				},
			},
		},
		nil,
		g.ContainerName,
	)
	if err != nil {
		return err
	}
	defer func() {
		stopHealthCheck <- struct{}{}

		g.Client.ContainerStop(ctx, resp.ID, durationPointer(30*time.Second))
		g.Client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	err = g.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	waitUntilHealthy := make(chan struct{})

	go func() {
		prevState := ""
		interval := time.NewTicker(1 * time.Second)

		for range interval.C {
			select {
			case <-stopHealthCheck:
				return
			default:
			}

			inspect, err := g.Client.ContainerInspect(ctx, resp.ID)
			if err != nil {
				panic(err)
			}

			if prevState != inspect.State.Health.Status {
				fmt.Println("STATUS CHANGE:", inspect.State.Health.Status)
				prevState = inspect.State.Health.Status

				if inspect.State.Health.Status == "healthy" {
					for _, l := range inspect.State.Health.Log {
						fmt.Println(l.Output)
					}

					waitUntilHealthy <- struct{}{}
				}
			}
		}
	}()

	// create the directory path for the go app
	err = dockerExec(ctx, g.Client, resp.ID, []string{"mkdir", "-p", g.AppDir})
	if err != nil {
		return err
	}

	tarContent := bytes.Buffer{}

	err = Tar(g.AppDir, &tarContent)
	if err != nil {
		return err
	}

	err = g.Client.CopyToContainer(ctx, resp.ID, g.AppDir, &tarContent, types.CopyToContainerOptions{})
	if err != nil {
		return err
	}

	fmt.Println("adding curl...")

	err = dockerExec(ctx, g.Client, resp.ID, []string{"apk", "update"})
	if err != nil {
		return err
	}

	err = dockerExec(ctx, g.Client, resp.ID, []string{"apk", "add", "curl"})
	if err != nil {
		return err
	}

	fmt.Println("building go app...")

	// build the go app inside the container
	err = dockerExec(ctx, g.Client, resp.ID, []string{"go", "build", path.Join(g.AppDir, g.BuildDir)})
	if err != nil {
		return err
	}

	fmt.Println("starting go app...")

	// run the go app inside the container
	err = dockerExec(ctx, g.Client, resp.ID, []string{"./" + path.Base(g.BuildDir)})
	if err != nil {
		return err
	}

	fmt.Println("go app is running.")

	timeout := time.NewTimer(1 * time.Minute)

	select {
	case <-waitUntilHealthy:
		// do nothing
	case <-timeout.C:
		inspect, err := g.Client.ContainerInspect(ctx, resp.ID)
		if err != nil {
			panic(err)
		}

		numOfLogs := len(inspect.State.Health.Log)
		lastHealthLog := ""

		if numOfLogs > 0 {
			lastHealthLog = inspect.State.Health.Log[numOfLogs-1].Output
		}

		return fmt.Errorf("timed out waiting for container to be healthy, the last healtcheck error was: %s", lastHealthLog)
	}

	fmt.Println("successfully created goapp container")

	return f()
}
