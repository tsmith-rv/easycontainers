package easycontainers

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
)

const (
	// ServiceSQS ...
	ServiceSQS = "sqs"
)

type sqsQueue struct {
	Name     string
	Messages []string
}

// Localstack is the container for localstack/localstack.
//
// SQS is a collection of queues and their messages to be created when the container initializes.
//
// Services is a list of services to start when the container starts up. You have to explicitly choose
// which to start because it is really hard on your system to start them all if you aren't using them.
//
// Environment is a map of environment variables to create in the container.
type Localstack struct {
	ContainerName string
	Port          int
	SQS           []sqsQueue
	Services      []string
	Environment   map[string]string
}

// NewLocalstack returns a new instance of Localstack and the port it will be using, which is
// a randomly selected number between 5000-6000.
//
// services is a list of services to start up when the container starts up.
//
// Conflicts are possible because it doesn't check if the port is already allocated.
func NewLocalstack(name string, services ...string) (r *Localstack, port int) {
	port = 5000 + rand.Intn(1000)

	return &Localstack{
		ContainerName: "localstack-" + name,
		Port:          port,
		Services:      services,
	}, port
}

// NewLocalstackWithPort returns a new instance of MySQL using the specified port.
func NewLocalstackWithPort(name string, port int, services ...string) *Localstack {
	return &Localstack{
		ContainerName: "localstack-" + name,
		Port:          port,
		Services:      services,
	}
}

// AddSQSQueue adds a queue and it's messages to be created when the container starts up.
func (l *Localstack) AddSQSQueue(name string, messages []string) *Localstack {
	l.SQS = append(l.SQS, sqsQueue{
		Name:     name,
		Messages: messages,
	})

	return l
}

// Container spins up the localstack container and runs. When the method exits, the
// container is stopped and removed.
func (l *Localstack) Container(f func() error) error {
	CleanupContainer(l.ContainerName) // catch containers that previous cleanup missed
	defer CleanupContainer(l.ContainerName)

	var cmdList []*exec.Cmd

	runContainerCmd := exec.Command(
		"docker",
		"run",
		"--rm",
		"-p",
		fmt.Sprintf("%d:4576", l.Port),
		"--name",
		l.ContainerName,
		"-d",
		"-e",
		fmt.Sprintf("SERVICES=%s", strings.Join(l.Services, ",")),
		"-e",
		"AWS_SECRET_ACCESS_KEY=guest",
		"-e",
		"AWS_ACCESS_KEY_ID=guest",
		"localstack/localstack",
	)

	cmdList = append(cmdList, runContainerCmd)

	installAWSCmd := cmdForContainer(
		l.ContainerName,
		exec.Command(
			"pip",
			"install",
			"awscli",
			"--upgrade",
			"--user",
		),
	)
	cmdList = append(cmdList, installAWSCmd)

	waitForInitializeCmd := strCmdForContainer(
		l.ContainerName,
		fmt.Sprintf(
			"until (/root/.local/bin/aws --region us-east-1 --endpoint-url=http://host.docker.internal:%d sqs list-queues) do echo 'waiting for localstack to be up'; sleep 1; done",
			l.Port,
		),
	)
	cmdList = append(cmdList, waitForInitializeCmd)

	for _, queue := range l.SQS {
		cmdList = append(cmdList, cmdForContainer(
			l.ContainerName,
			exec.Command(
				"/root/.local/bin/aws",
				"--endpoint-url",
				fmt.Sprintf("http://host.docker.internal:%d", l.Port),
				"sqs",
				"create-queue",
				"--queue-name",
				queue.Name,
				"--region",
				"us-east-1",
			),
		))

		for _, msg := range queue.Messages {
			cmdList = append(cmdList, cmdForContainer(
				l.ContainerName,
				exec.Command(
					"/root/.local/bin/aws",
					"--region",
					"us-east-1",
					"--endpoint-url",
					fmt.Sprintf("http://host.docker.internal:%d", l.Port),
					"sqs",
					"send-message",
					"--queue-url",
					fmt.Sprintf("http://host.docker.internal:%d/queue/%s", l.Port, queue.Name),
					"--message-body",
					fmt.Sprintf(`"%s"`, msg),
				),
			))
		}
	}

	for _, c := range cmdList {
		err := RunCommandWithTimeout(c)
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully created localstack container")

	return f()
}
