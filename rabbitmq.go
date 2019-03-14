package easycontainers

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"io"
	"os"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
)

const (
	rabbitmqadmin = "rabbitmqadmin"

	ExchangeTypeDirect = "direct"
	ExchangeTypeFanout = "fanout"
)

// RabbitMQ is a container using the official RabbitMQ:management docker image, which already
// has rabbitmqadmin installed on startup.
type RabbitMQ struct {
	Client        *client.Client
	ContainerName string
	Port          int
	Vhosts        []Vhost
	Exchanges     []Exchange
	Queues        []Queue
	Bindings      []QueueBinding
}

// Vhost is a RabbitMQ Virtual Host
type Vhost struct {
	Name string
}

// Exchange is a RabbitMQ Exchange
type Exchange struct {
	Name  string
	Type  string
	Vhost *Vhost
}

// Queue is a RabbitMQ Queue
type Queue struct {
	Name    string
	Durable bool
	Vhost   *Vhost
}

// QueueBinding is a RabbitMQ Binding between an Exchange (source) and Queue (destination)
type QueueBinding struct {
	Source      Exchange
	Destination Queue
	RoutingKey  string
	Vhost       *Vhost
}

// NewRabbitMQ returns a new instance of RabbitMQ and the port it will be using.
func NewRabbitMQ(name string) (r *RabbitMQ, port int) {
	port, err := getFreePort()
	if err != nil {
		panic(err)
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &RabbitMQ{
		ContainerName: prefix + "rabbit-" + name,
		Port:          port,
		Client:        c,
	}, port
}

// NewRabbitMQWithPort returns a new instance of RabbitMQ using the specified port.
func NewRabbitMQWithPort(name string, port int) *RabbitMQ {
	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &RabbitMQ{
		ContainerName: prefix + "rabbit-" + name,
		Port:          port,
		Client:        c,
	}
}

// Container spins up the rabbitmq container and runs. When the method exits, the
// container is stopped and removed.
//
// The RabbitMQ components will be created in the following order:
// Vhosts -> Exchanges -> Queues -> Bindings
func (r *RabbitMQ) Container(f func() error) error {
	ctx := context.Background()
	reader, err := r.Client.ImagePull(ctx, "docker.io/library/rabbitmq:management-alpine", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	stopHealthCheck := make(chan struct{})

	resp, err := r.Client.ContainerCreate(
		ctx,
		&container.Config{
			Image: "rabbitmq:management-alpine",
			Healthcheck: &container.HealthConfig{
				Test:     []string{"CMD-SHELL", "until $(rabbitmqadmin -q list queues); do echo 'waiting for RabbitMQ container to be up'; sleep 1; done"},
				Interval: 5 * time.Second,
				Timeout:  1 * time.Minute,
			},
		},
		&container.HostConfig{
			PortBindings: nat.PortMap{
				"5672/tcp": []nat.PortBinding{
					{
						HostIP:   "0.0.0.0",
						HostPort: strconv.Itoa(r.Port),
					},
				},
			},
		},
		nil,
		r.ContainerName,
	)
	if err != nil {
		return err
	}
	defer func() {
		stopHealthCheck <- struct{}{}

		r.Client.ContainerStop(ctx, resp.ID, durationPointer(30*time.Second))
		r.Client.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	err = r.Client.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
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

			inspect, err := r.Client.ContainerInspect(ctx, resp.ID)
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

	timeout := time.NewTimer(1 * time.Minute)

	select {
	case <-waitUntilHealthy:
		// do nothing
	case <-timeout.C:
		inspect, err := r.Client.ContainerInspect(ctx, resp.ID)
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

	for _, x := range r.Vhosts {
		err = dockerExec(ctx, r.Client, resp.ID, x.CreateCommand())
		if err != nil {
			return err
		}
	}

	for _, x := range r.Exchanges {
		err = dockerExec(ctx, r.Client, resp.ID, x.CreateCommand())
		if err != nil {
			return err
		}
	}

	for _, x := range r.Queues {
		err = dockerExec(ctx, r.Client, resp.ID, x.CreateCommand())
		if err != nil {
			return err
		}
	}

	for _, x := range r.Bindings {
		err = dockerExec(ctx, r.Client, resp.ID, x.CreateCommand())
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully created rabbitmq container")

	return f()
}

// AddVhosts adds the specified Vhosts to be created when the container starts.
//
// Vhosts are created before anything else when the container starts up.
func (r *RabbitMQ) AddVhosts(v ...Vhost) *RabbitMQ {
	r.Vhosts = append(r.Vhosts, v...)

	return r
}

// AddExchanges adds the specified Exchanges to be created when the container starts.
//
// Exchanges are created after the Vhosts.
func (r *RabbitMQ) AddExchanges(e ...Exchange) *RabbitMQ {
	r.Exchanges = append(r.Exchanges, e...)

	return r
}

// AddQueue adds the specified Queue to be created when the container starts.
//
// Queues are created after Vhosts and Exchanges.
func (r *RabbitMQ) AddQueue(q ...Queue) *RabbitMQ {
	r.Queues = append(r.Queues, q...)

	return r
}

// AddBinding adds the specified Binding to be created when the container starts.
//
// Bindings are created after Vhosts, Exchanges, and Queues.
func (r *RabbitMQ) AddBinding(b ...QueueBinding) *RabbitMQ {
	r.Bindings = append(r.Bindings, b...)

	return r
}

// CreateCommand returns a command for creating the Vhost from the command line.
func (v *Vhost) CreateCommand() []string {
	return []string{
		rabbitmqadmin,
		"declare",
		"vhost",
		fmt.Sprintf("name=%s", v.Name),
	}
}

// CreateCommand returns a command for creating the Exchange from the command line.
func (e *Exchange) CreateCommand() []string {
	args := []string{rabbitmqadmin}

	if e.Vhost != nil {
		args = append(
			args,
			"--vhost",
			e.Vhost.Name,
		)
	}

	args = append(
		args,
		"declare",
		"exchange",
		fmt.Sprintf("name=%s", e.Name),
		fmt.Sprintf("type=%s", e.Type),
	)

	return args
}

// CreateCommand returns a command for creating the Queue from the command line.
func (q *Queue) CreateCommand() []string {
	args := []string{rabbitmqadmin}

	if q.Vhost != nil {
		args = append(
			args,
			"--vhost",
			q.Vhost.Name,
		)
	}

	args = append(
		args,
		[]string{
			"declare",
			"queue",
			fmt.Sprintf("name=%s", q.Name),
			fmt.Sprintf("durable=%t", q.Durable),
		}...,
	)

	return args
}

// CreateCommand returns a command for creating the Binding from the command line.
func (q *QueueBinding) CreateCommand() []string {
	args := []string{rabbitmqadmin}

	if q.Vhost != nil {
		args = append(
			args,
			"--vhost",
			q.Vhost.Name,
		)
	}

	args = append(
		args,
		[]string{
			"declare",
			"binding",
			fmt.Sprintf("source=%s", q.Source.Name),
			"destination_type=queue",
			fmt.Sprintf("destination=%s", q.Destination.Name),
			fmt.Sprintf("routing_key=%s", q.RoutingKey),
		}...,
	)

	return args
}
