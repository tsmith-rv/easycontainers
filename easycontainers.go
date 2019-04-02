package easycontainers

import (
	"archive/tar"
	"compress/gzip"
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

	"bytes"
	"io"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/client"
)

const prefix = "easycontainers-"

var (
	getFreePortLock = &sync.Mutex{}
	allocatedPorts  = map[int]struct{}{}
)

type containerInfo struct {
	Ctx         context.Context
	Client      *client.Client
	ContainerID string
}

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

// Tar takes a source and variable writers and walks 'source' writing each file
// found to the tar writer; the purpose for accepting multiple writers is to allow
// for multiple outputs (for example a file, or md5 hash)
//
// Adapted from https://gist.githubusercontent.com/sdomino/e6bc0c98f87843bc26bb/raw/76e09bb99fc8ff3e9b8c1630008d4829d6b46320/targz.go
func Tar(src string, writers ...io.Writer) error {
	// ensure the src actually exists before trying to tar it
	if _, err := os.Stat(src); err != nil {
		return fmt.Errorf("Unable to tar files - %v", err.Error())
	}

	mw := io.MultiWriter(writers...)

	gzw := gzip.NewWriter(mw)
	defer gzw.Close()

	tw := tar.NewWriter(gzw)
	defer tw.Close()

	// walk path
	return filepath.Walk(src, func(file string, fi os.FileInfo, err error) error {

		// return on any error
		if err != nil {
			return err
		}

		// create a new dir/file header
		header, err := tar.FileInfoHeader(fi, fi.Name())
		if err != nil {
			return err
		}

		// update the name to correctly reflect the desired destination when untaring
		header.Name = strings.TrimPrefix(strings.Replace(file, src, "", -1), string(filepath.Separator))

		// write the header
		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		// return on non-regular files (thanks to [kumo](https://medium.com/@komuw/just-like-you-did-fbdd7df829d3) for this suggested update)
		if !fi.Mode().IsRegular() {
			return nil
		}

		// open files for taring
		f, err := os.Open(file)
		if err != nil {
			return err
		}

		// copy file data into tar writer
		if _, err := io.Copy(tw, f); err != nil {
			return err
		}

		// manually close here after each file operation; defering would cause each file close
		// to wait until all operations have completed.
		f.Close()

		return nil
	})
}

func dockerExec(ctx context.Context, client *client.Client, containerID string, cmd []string) error {
	e, err := client.ContainerExecCreate(ctx, containerID, types.ExecConfig{
		Detach:       true,
		Tty:          false,
		AttachStderr: true,
		Cmd:          cmd,
	})
	if err != nil {
		return err
	}

	attach, err := client.ContainerExecAttach(ctx, e.ID, types.ExecConfig{
		AttachStderr: true,
		Detach:       false,
		Tty:          false,
	})
	if err != nil {
		return err
	}
	defer attach.Close()

	b := bytes.Buffer{}
	_, err = io.Copy(&b, attach.Reader)
	if err != nil {
		return err
	}

	inspect, err := client.ContainerExecInspect(ctx, e.ID)
	if err != nil {
		return err
	}

	if inspect.ExitCode != 0 {
		return errors.New(b.String())
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
