package easycontainers

import (
	"fmt"
	"math/rand"
	"os/exec"
	"strings"
	"sync"
	"time"
)

const (
	ServiceSQS             = "sqs"
	ServiceAPIGateway      = "apigateway"
	ServiceKinesis         = "kinesis"
	ServiceS3              = "s3"
	ServiceDynamoDB        = "dynamodb"
	ServiceDynamoDBStreams = "dynamodbstreams"
	ServiceElasticsearch   = "elasticsearch"
	ServiceFirehose        = "firehose"
	ServiceLambda          = "lambda"
	ServiceSNS             = "sns"
	ServiceRedshift        = "redshift"
	ServiceES              = "es"
	ServiceSES             = "ses"
	ServiceRoute53         = "route53"
	ServiceCloudformation  = "cloudformation"
	ServiceCloudwatch      = "cloudwatch"
	ServiceSSM             = "ssm"
	ServiceSecretsManager  = "secretsmanager"
)

var ports = map[string]int{
	ServiceSQS:             4576,
	ServiceAPIGateway:      4567,
	ServiceKinesis:         4568,
	ServiceS3:              4572,
	ServiceDynamoDB:        4569,
	ServiceDynamoDBStreams: 4570,
	ServiceElasticsearch:   4571,
	ServiceFirehose:        4573,
	ServiceLambda:          4574,
	ServiceSNS:             4575,
	ServiceRedshift:        4577,
	ServiceES:              4578,
	ServiceSES:             4579,
	ServiceRoute53:         4580,
	ServiceCloudformation:  4581,
	ServiceCloudwatch:      4582,
	ServiceSSM:             4583,
	ServiceSecretsManager:  4584,
}

var initializations = map[string]string{
	ServiceSQS:             "sqs list-queues",
	ServiceAPIGateway:      "apigateway get-api-keys",
	ServiceKinesis:         "kinesis list-streams",
	ServiceS3:              "s3 ls",
	ServiceDynamoDB:        "dynamodb list-tables",
	ServiceDynamoDBStreams: "dynamodbstreams list-streams",
	ServiceFirehose:        "firehose list-delivery-streams",
	ServiceLambda:          "lambda list-functions",
	ServiceSNS:             "sns list-topics",
	ServiceRedshift:        "redshift describe-tags",
	ServiceES:              "es list-domain-names",
	ServiceSES:             "ses list-identities",
	ServiceRoute53:         "route53 list-health-checks",
	ServiceCloudformation:  "cloudformation describe-stacks",
	ServiceCloudwatch:      "cloudwatch describe-alarms",
	ServiceSSM:             "ssm list-commands",
	ServiceSecretsManager:  "secretsmanager get-random-password",
}

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
	SQS           []sqsQueue
	Services      []string
	PortBindings  map[string]string
	Environment   map[string]string
}

// NewLocalstack returns a new instance of Localstack and the port it will be using, which is
// a randomly selected number between 5000-6000.
//
// services is a list of services to start up when the container starts up.
//
// Conflicts are possible because it doesn't check if the port is already allocated.
func NewLocalstack(name string, services ...string) (r *Localstack, portMap map[string]int) {
	portMap = make(map[string]int)
	portBindings := make(map[string]string)

	for _, s := range services {
		port := 5000 + rand.Intn(1000)
		portMap[s] = port
		portBindings[s] = fmt.Sprintf("%d:%d", port, ports[s])
	}

	return &Localstack{
		ContainerName: prefix + "localstack-" + name,
		PortBindings:  portBindings,
		Services:      services,
	}, portMap
}

// NewLocalstackWithPortMap returns a new instance of localstack using the specified ports for services
func NewLocalstackWithPortMap(name string, portMap map[string]int, services ...string) *Localstack {
	portBindings := make(map[string]string)

	for _, s := range services {
		port := 5000 + rand.Intn(1000)
		portMap[s] = port
		portBindings[s] = fmt.Sprintf("%d:%d", port, ports[s])
	}

	return &Localstack{
		ContainerName: prefix + "localstack-" + name,
		PortBindings:  portBindings,
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

	var runArgs []string
	{
		runArgs = []string{
			"run",
			"--rm",
			"--name",
			l.ContainerName,
			"-d",
		}

		for _, binding := range l.PortBindings {
			runArgs = append(runArgs, "-p", binding)
		}

		runArgs = append(runArgs, []string{
			"-e",
			fmt.Sprintf("SERVICES=%s", strings.Join(l.Services, ",")),
			"-e",
			"AWS_SECRET_ACCESS_KEY=guest",
			"-e",
			"AWS_ACCESS_KEY_ID=guest",
			"localstack/localstack",
		}...)
	}

	runContainerCmd := exec.Command(
		"docker",
		runArgs...,
	)

	err := RunCommandWithTimeout(runContainerCmd, 1*time.Minute)
	if err != nil {
		return err
	}

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

	err = RunCommandWithTimeout(installAWSCmd, 1*time.Minute)
	if err != nil {
		return err
	}

	errs := make(chan error, len(l.Services))
	wg := &sync.WaitGroup{}

	for _, s := range l.Services {
		if _, exists := initializations[s]; !exists {
			continue
		}

		wg.Add(1)

		go func(s string) {
			defer wg.Done()

			waitForInitializeCmd := strCmdForContainer(
				l.ContainerName,
				fmt.Sprintf(
					"until (/root/.local/bin/aws --region us-east-1 --endpoint-url=http://localhost:%d %s) do echo 'waiting for localstack - %s to be up'; sleep 1; done",
					ports[s],
					initializations[s],
					s,
				),
			)

			err = RunCommandWithTimeout(waitForInitializeCmd, 1*time.Minute)
			if err != nil {
				errs <- err
				return
			}
		}(s)
	}

	wg.Wait()

	select {
	case err := <-errs:
		return err
	default:
	}

	for _, queue := range l.SQS {
		createQueueCmd := cmdForContainer(
			l.ContainerName,
			exec.Command(
				"/root/.local/bin/aws",
				"--endpoint-url",
				"http://localhost:4576",
				"sqs",
				"create-queue",
				"--queue-name",
				queue.Name,
				"--region",
				"us-east-1",
			),
		)
		err = RunCommandWithTimeout(createQueueCmd, 5*time.Second)
		if err != nil {
			return err
		}

		for _, msg := range queue.Messages {
			sendMsgCmd := cmdForContainer(
				l.ContainerName,
				exec.Command(
					"/root/.local/bin/aws",
					"--region",
					"us-east-1",
					"--endpoint-url",
					"http://localhost:4576",
					"sqs",
					"send-message",
					"--queue-url",
					fmt.Sprintf("http://localhost:4576/queue/%s", queue.Name),
					"--message-body",
					fmt.Sprintf(`"%s"`, msg),
				),
			)
			err = RunCommandWithTimeout(sendMsgCmd, 5*time.Second)
			if err != nil {
				return err
			}
		}
	}

	fmt.Println("successfully created localstack container")

	return f()
}
