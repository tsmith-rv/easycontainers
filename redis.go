package easycontainers

import (
	"context"
	"fmt"
	"time"

	"strconv"

	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

// Redis is a containerized version of the specified Go application.
type Redis struct {
	Client        *client.Client
	ContainerName string
	Port          int
}

// NewRedis returns a new instance of Redis and the port it will be using.
func NewRedis(name string) (r *Redis, port int) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &Redis{
		Client:        c,
		ContainerName: prefix + "-redis-" + name,
		Port:          port,
	}, port
}

// NewRedisWithPort returns a new instance of Redis using the specified port.
func NewRedisWithPort(name string, port int) *Redis {
	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &Redis{
		Client:        c,
		ContainerName: prefix + "-redis-" + name,
		Port:          port,
	}
}

// Container spins up the application container and runs. When the method exits, the
// container is stopped and removed.
func (redis *Redis) Container(f func() error) error {
	ctx := context.Background()
	reader, err := redis.Client.ImagePull(ctx, "docker.io/library/redis:latest", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	resp, err := redis.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: "redis:latest",
			Tty:   true,
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"6379/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: strconv.Itoa(redis.Port),
					},
				},
			},
		},
		nil,
		redis.ContainerName,
	)
	if err != nil {
		return err
	}
	defer func() {
		redis.Client.ContainerStop(ctx, resp.ID, durationPointer(30*time.Second))
		redis.Client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	err = redis.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	fmt.Println("successfully created Redis container")

	return f()
}
