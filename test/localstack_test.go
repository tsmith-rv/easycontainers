package test

import (
	"testing"

	"strconv"

	"sync"

	"fmt"

	"github.com/stretchr/testify/assert"
	"github.com/tsmith-rv/easycontainers"
)

func Test_Localstack_Container(t *testing.T) {
	localstackContainer, ports := easycontainers.NewLocalstack(
		"Test_Localstack_Container",
		easycontainers.ServiceSQS,
		easycontainers.ServiceAPIGateway,
		easycontainers.ServiceKinesis,
		easycontainers.ServiceS3,
		easycontainers.ServiceDynamoDB,
		easycontainers.ServiceDynamoDBStreams,
		easycontainers.ServiceElasticsearch,
		easycontainers.ServiceFirehose,
		easycontainers.ServiceLambda,
		easycontainers.ServiceSNS,
		easycontainers.ServiceRedshift,
		easycontainers.ServiceES,
		easycontainers.ServiceSES,
		easycontainers.ServiceRoute53,
		easycontainers.ServiceCloudformation,
		easycontainers.ServiceCloudwatch,
		easycontainers.ServiceSSM,
		easycontainers.ServiceSecretsManager,
	)
	err := localstackContainer.Container(func(queues []easycontainers.SQSQueue, lambdas []easycontainers.Lambda) error {
		return nil
	})
	if !assert.NoError(t, err) {
		return
	}

	for _, p := range ports {
		isFree, err := isPortFree(p)
		if !assert.NoError(t, err) {
			return
		}

		if !assert.True(t, isFree, "port %d should now be available, but isn't", p) {
			return
		}
	}
}

func Test_Localstack_SQS_SendMessage(t *testing.T) {
	localstackContainer, ports := easycontainers.NewLocalstack(
		"Test_Localstack_SQS_SendMessage",
		easycontainers.ServiceSQS,
		easycontainers.ServiceAPIGateway,
		easycontainers.ServiceKinesis,
		easycontainers.ServiceS3,
		easycontainers.ServiceDynamoDB,
		easycontainers.ServiceDynamoDBStreams,
		easycontainers.ServiceElasticsearch,
		easycontainers.ServiceFirehose,
		easycontainers.ServiceLambda,
		easycontainers.ServiceSNS,
		easycontainers.ServiceRedshift,
		easycontainers.ServiceES,
		easycontainers.ServiceSES,
		easycontainers.ServiceRoute53,
		easycontainers.ServiceCloudformation,
		easycontainers.ServiceCloudwatch,
		easycontainers.ServiceSSM,
		easycontainers.ServiceSecretsManager,
	)

	localstackContainer.
		AddSQSQueue("queue1").
		AddSQSQueue("queue2").
		AddSQSQueue("queue3")

	err := localstackContainer.Container(func(queues []easycontainers.SQSQueue, lambdas []easycontainers.Lambda) error {
		var wg sync.WaitGroup
		wg.Add(75)

		for i := 0; i < 25; i++ {
			go func() {
				defer wg.Done()

				err := queues[0].SendMessage(localstackContainer.ContainerName, strconv.Itoa(i))
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		for i := 0; i < 25; i++ {
			go func() {
				defer wg.Done()

				err := queues[1].SendMessage(localstackContainer.ContainerName, strconv.Itoa(i))
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		for i := 0; i < 25; i++ {
			go func() {
				defer wg.Done()

				err := queues[2].SendMessage(localstackContainer.ContainerName, strconv.Itoa(i))
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		wg.Wait()

		return nil
	})
	if !assert.NoError(t, err) {
		return
	}

	fmt.Println("checking", len(ports), "ports for closure")

	for _, p := range ports {
		isFree, err := isPortFree(p)
		if !assert.NoError(t, err) {
			return
		}

		if !assert.True(t, isFree, "port %d should now be available, but isn't", p) {
			return
		}
	}
}

func Test_Localstack_Lambda_SendPayload(t *testing.T) {
	localstackContainer, ports := easycontainers.NewLocalstack(
		"Test_Localstack_Lambda_SendPayload",
		easycontainers.ServiceSQS,
		easycontainers.ServiceAPIGateway,
		easycontainers.ServiceKinesis,
		easycontainers.ServiceS3,
		easycontainers.ServiceDynamoDB,
		easycontainers.ServiceDynamoDBStreams,
		easycontainers.ServiceElasticsearch,
		easycontainers.ServiceFirehose,
		easycontainers.ServiceLambda,
		easycontainers.ServiceSNS,
		easycontainers.ServiceRedshift,
		easycontainers.ServiceES,
		easycontainers.ServiceSES,
		easycontainers.ServiceRoute53,
		easycontainers.ServiceCloudformation,
		easycontainers.ServiceCloudwatch,
		easycontainers.ServiceSSM,
		easycontainers.ServiceSecretsManager,
	)

	localstackContainer.
		AddLambda("function1", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddLambda("function2", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddLambda("function3", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip")

	err := localstackContainer.Container(func(queues []easycontainers.SQSQueue, lambdas []easycontainers.Lambda) error {
		var wg sync.WaitGroup
		wg.Add(9)

		for i := 0; i < 3; i++ {
			go func() {
				defer wg.Done()

				err := lambdas[0].SendPayload(localstackContainer.ContainerName, map[string]interface{}{
					"What is your name?": "tsmith-rv",
					"How old are you?":   i,
				})
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		for i := 0; i < 3; i++ {
			go func() {
				defer wg.Done()

				err := lambdas[1].SendPayload(localstackContainer.ContainerName, map[string]interface{}{
					"What is your name?": "tsmith-rv",
					"How old are you?":   i,
				})
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		for i := 0; i < 3; i++ {
			go func() {
				defer wg.Done()

				err := lambdas[2].SendPayload(localstackContainer.ContainerName, map[string]interface{}{
					"What is your name?": "tsmith-rv",
					"How old are you?":   i,
				})
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		wg.Wait()

		return nil
	})
	if !assert.NoError(t, err) {
		return
	}

	fmt.Println("checking", len(ports), "ports for closure")

	for _, p := range ports {
		isFree, err := isPortFree(p)
		if !assert.NoError(t, err) {
			return
		}

		if !assert.True(t, isFree, "port %d should now be available, but isn't", p) {
			return
		}
	}
}

func Test_Localstack_Lambda_SendPayload_BadPayload(t *testing.T) {
	localstackContainer, ports := easycontainers.NewLocalstack(
		"Test_Localstack_Lambda_SendPayload_BadPayload",
		easycontainers.ServiceSQS,
		easycontainers.ServiceAPIGateway,
		easycontainers.ServiceKinesis,
		easycontainers.ServiceS3,
		easycontainers.ServiceDynamoDB,
		easycontainers.ServiceDynamoDBStreams,
		easycontainers.ServiceElasticsearch,
		easycontainers.ServiceFirehose,
		easycontainers.ServiceLambda,
		easycontainers.ServiceSNS,
		easycontainers.ServiceRedshift,
		easycontainers.ServiceES,
		easycontainers.ServiceSES,
		easycontainers.ServiceRoute53,
		easycontainers.ServiceCloudformation,
		easycontainers.ServiceCloudwatch,
		easycontainers.ServiceSSM,
		easycontainers.ServiceSecretsManager,
	)

	localstackContainer.
		AddLambda("function1", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddLambda("function2", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddLambda("function3", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip")

	err := localstackContainer.Container(func(queues []easycontainers.SQSQueue, lambdas []easycontainers.Lambda) error {
		for _, lambda := range lambdas {
			err := lambda.SendPayload(localstackContainer.ContainerName, map[string]interface{}{
				"What is your name?": 3, // value should be a string, not an int
				"How old are you?":   33,
			})
			if !assert.Error(t, err) {
				return nil
			}
		}

		return nil
	})
	if !assert.NoError(t, err) {
		return
	}

	fmt.Println("checking", len(ports), "ports for closure")

	for _, p := range ports {
		isFree, err := isPortFree(p)
		if !assert.NoError(t, err) {
			return
		}

		if !assert.True(t, isFree, "port %d should now be available, but isn't", p) {
			return
		}
	}
}
