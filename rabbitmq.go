package easycontainers

import (
	"fmt"
	"math/rand"
	"os/exec"
	"time"
)

const (
	rabbitmqadmin = "rabbitmqadmin"

	ExchangeTypeDirect = "direct"
	ExchangeTypeFanout = "fanout"
)

// RabbitMQ is a container using the official RabbitMQ:management docker image, which already
// has rabbitmqadmin installed on startup.
type RabbitMQ struct {
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

// NewRabbitMQ returns a new instance of RabbitMQ and the port it will be using, which is
// a randomly selected number between 5000-6000.
//
// Conflicts are possible because it doesn't check if the port is already allocated.
func NewRabbitMQ(name string) (r *RabbitMQ, port int) {
	port = 5000 + rand.Intn(1000)

	return &RabbitMQ{
		ContainerName: prefix + "rabbit-" + name,
		Port:          port,
	}, port
}

// NewRabbitMQWithPort returns a new instance of RabbitMQ using the specified port.
func NewRabbitMQWithPort(name string, port int) *RabbitMQ {
	return &RabbitMQ{
		ContainerName: prefix + "rabbit-" + name,
		Port:          port,
	}
}

// Container spins up the mysql container and runs. When the method exits, the
// container is stopped and removed.
//
// The RabbitMQ components will be created in the following order:
// Vhosts -> Exchanges -> Queues -> Bindings
func (r *RabbitMQ) Container(f func() error) error {
	CleanupContainer(r.ContainerName)
	defer CleanupContainer(r.ContainerName)

	var cmdList []*exec.Cmd

	runContainerCmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-p",
		fmt.Sprintf("%d:5672", r.Port),
		"--name",
		r.ContainerName,
		"-d",
		"rabbitmq:management-alpine",
	)
	cmdList = append(cmdList, runContainerCmd)

	waitForInitializeCmd := strCmdForContainer(
		r.ContainerName,
		"until $(rabbitmqadmin -q list queues); do echo 'waiting for RabbitMQ container to be up'; sleep 1; done",
	)
	cmdList = append(cmdList, waitForInitializeCmd)

	for _, x := range r.Vhosts {
		cmdList = append(
			cmdList,
			cmdForContainer(
				r.ContainerName,
				x.CreateCommand(),
			),
		)
	}

	for _, x := range r.Exchanges {
		cmdList = append(
			cmdList,
			cmdForContainer(
				r.ContainerName,
				x.CreateCommand(),
			),
		)
	}

	for _, x := range r.Queues {
		cmdList = append(
			cmdList,
			cmdForContainer(
				r.ContainerName,
				x.CreateCommand(),
			),
		)
	}

	for _, x := range r.Bindings {
		cmdList = append(
			cmdList,
			cmdForContainer(
				r.ContainerName,
				x.CreateCommand(),
			),
		)
	}

	for _, c := range cmdList {
		err := RunCommandWithTimeout(c, 1*time.Minute)
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
func (v *Vhost) CreateCommand() *exec.Cmd {
	return exec.Command(
		rabbitmqadmin,
		"declare",
		"vhost",
		fmt.Sprintf("name=%s", v.Name),
	)
}

// CreateCommand returns a command for creating the Exchange from the command line.
func (e *Exchange) CreateCommand() *exec.Cmd {
	var args []string

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

	return exec.Command(
		rabbitmqadmin,
		args...,
	)
}

// CreateCommand returns a command for creating the Queue from the command line.
func (q *Queue) CreateCommand() *exec.Cmd {
	var args []string

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

	return exec.Command(
		rabbitmqadmin,
		args...,
	)
}

// CreateCommand returns a command for creating the Binding from the command line.
func (q *QueueBinding) CreateCommand() *exec.Cmd {
	var args []string

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

	return exec.Command(
		rabbitmqadmin,
		args...,
	)
}
