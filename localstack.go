package easycontainers

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"path"
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

// SQSQueue is a queue in SQS in Localstack (who knew?)
type SQSQueue struct {
	Name string
}

// LambdaFunction is a lambda function in localstack (who knew?)
type LambdaFunction struct {
	FunctionName string
	Handler      string
	Zip          string
	Payloads     []string
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
	PortBindings  map[string]string
	Environment   map[string]string
}

// NewLocalstack returns a new instance of Localstack and the port it will be using.
func NewLocalstack(name string, services ...string) (r *Localstack, portMap map[string]int) {
	portMap = make(map[string]int)
	portBindings := make(map[string]string)

	for _, s := range services {
		port, err := getFreePort()
		if err != nil {
			panic(err)
		}

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
		portBindings[s] = fmt.Sprintf("%d:%d", portMap[s], ports[s])
	}

	return &Localstack{
		ContainerName: prefix + "localstack-" + name,
		PortBindings:  portBindings,
		Services:      services,
	}
}

// SendPayload sends the specified payload, marshaled into json.
//
// The json is used as a token in the command line, and doesn't work right with
// single quotes and stuff (for now), so please, don't use single quotes or special
// bash characters, OR YOU'RE GONNA HAVE A BAD TIME.
func (l *LambdaFunction) SendPayload(container string, payload map[string]interface{}) error {
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	str := "'" + string(payloadJSON) + "'"

	sendMsgCmd := cmdForContainer(
		container,
		exec.Command(
			"/root/.local/bin/aws",
			"--endpoint-url",
			"http://localhost:4574",
			"lambda",
			"invoke",
			"--region",
			"us-east-1",
			"--function-name",
			l.FunctionName,
			"output.out",
			"--payload",
			str,
		),
	)

	return RunCommandWithTimeout(sendMsgCmd, 1*time.Minute)
}

// CreateCommand returns a command for creating the lambda from the command line.
func (l *LambdaFunction) CreateCommand() *exec.Cmd {
	return exec.Command(
		"/root/.local/bin/aws",
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
		"fileb:///"+path.Base(l.Zip),
	)
}

// CreateCommand returns a command for creating the queue from the command line.
func (q *SQSQueue) CreateCommand() *exec.Cmd {
	return exec.Command(
		"/root/.local/bin/aws",
		"--endpoint-url",
		"http://localhost:4576",
		"sqs",
		"create-queue",
		"--queue-name",
		q.Name,
		"--region",
		"us-east-1",
	)
}

// SendMessage sends the specified message to the queue in the container
func (l *SQSQueue) SendMessage(container string, msg string) error {
	sendMsgCmd := cmdForContainer(
		container,
		exec.Command(
			"/root/.local/bin/aws",
			"--region",
			"us-east-1",
			"--endpoint-url",
			"http://localhost:4576",
			"sqs",
			"send-message",
			"--queue-url",
			fmt.Sprintf("http://localhost:4576/queue/%s", l.Name),
			"--message-body",
			fmt.Sprintf(`"%s"`, msg),
		),
	)

	return RunCommandWithTimeout(sendMsgCmd, 1*time.Minute)
}

// AddQueue adds a queue and it's messages to be created when the container starts up.
func (l *Localstack) AddQueue(name string) *Localstack {
	l.Queues = append(l.Queues, SQSQueue{
		Name: name,
	})

	return l
}

// AddFunction adds a LambdaFunction and it's payloads to be created when the container starts up
func (l *Localstack) AddFunction(functionName, handler, zip string) *Localstack {
	l.Functions = append(l.Functions, LambdaFunction{
		FunctionName: functionName,
		Handler:      handler,
		Zip:          zip,
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

		// if Functions is being run, mount the unix socket or Functions's won't run
		for _, s := range l.Services {
			if s == ServiceLambda {
				runArgs = append(runArgs, "-v", "/var/run/docker.sock:/var/run/docker.sock")
			}
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
			"-e",
			"LAMBDA_EXECUTOR=docker",
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

	for _, queue := range l.Queues {
		createQueueCmd := cmdForContainer(
			l.ContainerName,
			queue.CreateCommand(),
		)

		err = RunCommandWithTimeout(createQueueCmd, 5*time.Second)
		if err != nil {
			return err
		}
	}

	for _, lambda := range l.Functions {
		addStartupSQLFileCmd := exec.Command(
			"/bin/bash",
			"-c",
			fmt.Sprintf(
				`docker cp %s $(docker ps --filter="name=^/%s$" --format="{{.ID}}"):/`,
				path.Join(GoPath(), lambda.Zip),
				l.ContainerName,
			),
		)

		err = RunCommandWithTimeout(addStartupSQLFileCmd, 10*time.Second)
		if err != nil {
			return err
		}

		createLambdaCommand := cmdForContainer(
			l.ContainerName,
			lambda.CreateCommand(),
		)

		err = RunCommandWithTimeout(createLambdaCommand, 10*time.Second)
		if err != nil {
			return err
		}
	}

	fmt.Println("successfully created localstack container")

	return f()
}
