package easycontainers

import (
	"context"
	"errors"
	"math/rand"
	"strings"
	"time"

	"fmt"
	"os"

	"go/build"
	"net"
	"path/filepath"
	"sync"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const prefix = "easycontainers-"

var (
	getFreePortLock = &sync.Mutex{}
	allocatedPorts  = map[int]struct{}{}
)

func init() {
	// we random numbers for port generation
	rand.Seed(time.Now().UTC().UnixNano())

	// cleanup any oustanding containers with the easycontainers prefix
	err := CleanupAllContainers()
	if err != nil {
		panic(err)
	}

	err = WaitForCleanup()
	if err != nil {
		panic(err)
	}

	// cleanup any outstanding easycontainers files in temp
	filepath.Walk(os.TempDir(), func(path string, info os.FileInfo, err error) error {
		if strings.HasPrefix(info.Name(), prefix) {
			os.Remove(path)
		}

		return nil
	})
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
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	// only grab the containers created by easycontainers
	args := filters.NewArgs()
	args.Add("name", prefix)

	containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
		All:     true,
		Filters: args,
	})
	if err != nil {
		return err
	}

	for _, container := range containers {
		fmt.Println("Killing and Removing Container", container.ID[:10])

		if err := cli.ContainerRemove(ctx, container.ID, types.ContainerRemoveOptions{
			Force: true,
		}); err != nil {
			return err
		}
	}

	return err
}

// WaitForCleanup checks every second if there are any easycontainers containers still
// live, and exits when there aren't, or when the timeout occurrs -- whichever comes first
func WaitForCleanup() error {
	ctx := context.Background()
	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	var (
		interval = time.NewTicker(1 * time.Second)
		timeout  = time.NewTimer(1 * time.Minute)
	)

	for range interval.C {
		select {
		case <-timeout.C:
			return errors.New("timed out waiting for all easycontainers containers to get removed")
		default:
			// only grab the containers created by easycontainers
			args := filters.NewArgs()
			args.Add("name", "/"+prefix)

			containers, err := cli.ContainerList(ctx, types.ContainerListOptions{
				Filters: args,
			})
			if err != nil {
				return err
			}

			if len(containers) == 0 {
				return nil
			}
		}
	}

	return nil
}

func dockerExec(ctx context.Context, client *client.Client, containerID string, cmd []string) error {
	e, err := client.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Detach:       false,
		AttachStderr: true,
		AttachStdout: true,
		Cmd:          cmd,
	})
	if err != nil {
		return err
	}

	err = client.ContainerExecStart(ctx, e.ID, types.ExecStartCheck{
		Detach: false,
		Tty:    false,
	})
	if err != nil {
		return err
	}

	return nil
}

func getFreePort() (int, error) {
	getFreePortLock.Lock()
	defer getFreePortLock.Unlock()

	// because ports that get returned don't connect until the containers are
	// actually started, the same port can get returned for multiple containers
	// which causes issues, so if a port has already been returned at some point,
	// don't return it again, just check for another port up to 10 times
	for i := 0; i < 10; i++ {
		// this block is a code snippet from github.com/phayes/freeport
		addr, err := net.ResolveTCPAddr("tcp", "localhost:0")
		if err != nil {
			return 0, err
		}

		l, err := net.ListenTCP("tcp", addr)
		if err != nil {
			return 0, err
		}
		defer l.Close()

		port := l.Addr().(*net.TCPAddr).Port
		if _, exists := allocatedPorts[port]; !exists {
			allocatedPorts[port] = struct{}{}

			return port, nil
		}
	}

	return 0, errors.New("took too long to find free port")
}

func durationPointer(d time.Duration) *time.Duration {
	return &d
}
