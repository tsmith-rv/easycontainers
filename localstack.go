package easycontainers

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"encoding/json"

	"time"

	"bytes"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/go-connections/nat"
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

// SQSQueue is a queue in SQS in Localstack (who knew?)
type SQSQueue struct {
	Name      string
	container *containerInfo
}

// LambdaFunction is a lambda function in localstack (who knew?)
type LambdaFunction struct {
	FunctionName string
	Handler      string
	Zip          string
	Payloads     []string
	container    *containerInfo
}

// Localstack is the container for localstack/localstack.
//
// Queues is a collection of queues and their messages to be created when the container initializes.
//
// Services is a list of services to start when the container starts up. You have to explicitly choose
// which to start because it is really hard on your system to start them all if you aren't using them.
//
// Environment is a map of environment variables to create in the container.
type Localstack struct {
	ContainerName string
	Queues        []SQSQueue
	Functions     []LambdaFunction
	Services      []string
	PortBindings  map[string]int
	Environment   map[string]string
	container     *containerInfo
}

// NewLocalstack returns a new instance of Localstack and the port it will be using.
func NewLocalstack(name string, services ...string) (r *Localstack, portMap map[string]int) {
	// if no services are specified, startup all services
	if len(services) == 0 {
		for service := range ports {
			services = append(services, service)
		}
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	portMap = make(map[string]int)

	for _, s := range services {
		port, err := getFreePort()
		if err != nil {
			panic(err)
		}

		portMap[s] = port
	}

	return &Localstack{
		ContainerName: prefix + "localstack-" + name,
		PortBindings:  portMap,
		Services:      services,
		container: &containerInfo{
			Client: c,
		},
	}, portMap
}

// NewLocalstackWithPortMap returns a new instance of localstack using the specified ports for services
func NewLocalstackWithPortMap(name string, portMap map[string]int, services ...string) *Localstack {
	// if no services are specified, startup all services
	if len(services) == 0 {
		for service := range ports {
			services = append(services, service)
		}
	}

	c, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	return &Localstack{
		ContainerName: prefix + "localstack-" + name,
		PortBindings:  portMap,
		Services:      services,
		container: &containerInfo{
			Client: c,
		},
	}
}

// SendPayload sends the specified payload, marshaled into json.
//
// The json is used as a token in the command line, and doesn't work right with
// single quotes and stuff (for now), so please, don't use single quotes or special
// bash characters, OR YOU'RE GONNA HAVE A BAD TIME.
func (l *LambdaFunction) SendPayload(payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return dockerExec(
		l.container.Ctx,
		l.container.Client,
		l.container.ContainerID,
		[]string{
			"aws",
			"--endpoint-url",
			"http://localhost:4574",
			"--region",
			"us-east-1",
			"lambda",
			"invoke",
			"--function-name",
			l.FunctionName,
			"output.out",
			"--payload",
			string(payloadJSON),
		},
	)
}

// CreateCommand returns a command for creating the lambda from the command line.
func (l *LambdaFunction) CreateCommand() []string {
	return []string{
		"aws",
		"--endpoint-url",
		"http://localhost:4574",
		"lambda",
		"create-function",
		"--region",
		"us-east-1",
		"--function-name",
		l.FunctionName,
		"--handler",
		l.Handler,
		"--memory",
		"128",
		"--role",
		"r1",
		"--runtime",
		"go1.x",
		"--zip-file",
		"fileb:///" + path.Base(l.Zip),
	}
}

// CreateCommand returns a command for creating the queue from the command line.
func (q *SQSQueue) CreateCommand() []string {
	return []string{
		"aws",
		"--endpoint-url",
		"http://localhost:4576",
		"sqs",
		"create-queue",
		"--queue-name",
		q.Name,
		"--region",
		"us-east-1",
	}
}

// SendMessage sends the specified message to the queue in the container
func (q *SQSQueue) SendMessage(msg string) error {
	return dockerExec(
		q.container.Ctx,
		q.container.Client,
		q.container.ContainerID,
		[]string{
			"aws",
			"--region",
			"us-east-1",
			"--endpoint-url",
			"http://localhost:4576",
			"sqs",
			"send-message",
			"--queue-url",
			fmt.Sprintf("http://localhost:4576/queue/%s", q.Name),
			"--message-body",
			fmt.Sprintf(`"%s"`, msg),
		},
	)
}

// AddQueue adds a queue and it's messages to be created when the container starts up.
func (l *Localstack) AddQueue(name string) *Localstack {
	l.Queues = append(l.Queues, SQSQueue{
		Name:      name,
		container: l.container,
	})

	return l
}

// AddFunction adds a LambdaFunction and it's payloads to be created when the container starts up
func (l *Localstack) AddFunction(functionName, handler, zip string) *Localstack {
	l.Functions = append(l.Functions, LambdaFunction{
		FunctionName: functionName,
		Handler:      handler,
		Zip:          zip,
		container:    l.container,
	})

	return l
}

// Container spins up the localstack container and runs. When the method exits, the
// container is stopped and removed.
func (l *Localstack) Container(f func() error) error {
	dockerClient := l.container.Client

	ctx := context.Background()
	l.container.Ctx = ctx

	reader, err := dockerClient.ImagePull(ctx, "docker.io/localstack/localstack", types.ImagePullOptions{})
	if err != nil {
		return err
	}
	defer reader.Close()

	_, err = io.Copy(os.Stdout, reader)
	if err != nil {
		return err
	}

	dockerConfig := container.Config{
		AttachStdout: true,
		AttachStderr: true,
		Image:        "localstack/localstack",
		Env: []string{
			fmt.Sprintf("SERVICES=%s", strings.Join(l.Services, ",")),
			"AWS_SECRET_ACCESS_KEY=guest",
			"AWS_ACCESS_KEY_ID=guest",
			"LAMBDA_EXECUTOR=docker",
		},
	}

	portMap := nat.PortMap{}
	for service, port := range l.PortBindings {
		p := fmt.Sprintf("%d/tcp", ports[service])
		portMap[nat.Port(p)] = []nat.PortBinding{
			{
				HostIP:   "0.0.0.0",
				HostPort: fmt.Sprintf("%d/tcp", port),
			},
		}
	}

	hostConfig := container.HostConfig{
		PortBindings: portMap,

		// this mount is required for Lambda's to work (don't know why)
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeBind,
				Source: "/var/run/docker.sock",
				Target: "/var/run/docker.sock",
			},
		},
	}

	resp, err := dockerClient.ContainerCreate(
		ctx,
		&dockerConfig,
		&hostConfig,
		nil,
		l.ContainerName,
	)
	if err != nil {
		return err
	}
	defer func() {
		dockerClient.ContainerStop(ctx, resp.ID, durationPointer(30*time.Second))
		dockerClient.ContainerRemove(ctx, resp.ID, types.ContainerRemoveOptions{
			Force: true,
		})
	}()

	l.container.ContainerID = resp.ID

	err = dockerClient.ContainerStart(ctx, resp.ID, types.ContainerStartOptions{})
	if err != nil {
		return err
	}

	err = dockerExec(
		ctx,
		dockerClient,
		resp.ID,
		[]string{
			"pip",
			"install",
			"awscli",
			"--upgrade",
			"--user",
		},
	)
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

		go func() {
			defer wg.Done()

			err := dockerExec(
				ctx,
				dockerClient,
				resp.ID,
				[]string{
					"/bin/bash",
					"-c",
					fmt.Sprintf(
						"until (aws --region us-east-1 --endpoint-url=http://localhost:%d %s) do echo 'waiting for localstack - %s to be up'; sleep 1; done",
						ports[s],
						initializations[s],
						s,
					),
				},
			)
			if err != nil {
				errs <- err
			}
		}()
	}

	wg.Wait()

	select {
	case err := <-errs:
		return err
	default:
	}

	for _, queue := range l.Queues {
		err = dockerExec(ctx, dockerClient, resp.ID, queue.CreateCommand())
		if err != nil {
			return err
		}
	}

	for _, lambda := range l.Functions {
		filePath := path.Join(GoPath(), lambda.Zip)

		tarContent := bytes.Buffer{}

		err = Tar(filePath, &tarContent)
		if err != nil {
			return err
		}

		err = dockerClient.CopyToContainer(ctx, resp.ID, "/", &tarContent, types.CopyToContainerOptions{})
		if err != nil {
			return err
		}

		err = dockerExec(ctx, dockerClient, resp.ID, lambda.CreateCommand())
	}

	fmt.Println("successfully created localstack container")

	return f()
}
