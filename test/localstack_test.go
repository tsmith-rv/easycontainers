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
	err := localstackContainer.Container(func() error {
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

func Test_Localstack_Container_NoServicesSpecified(t *testing.T) {
	localstackContainer, ports := easycontainers.NewLocalstack(
		"Test_Localstack_Container_NoServicesSpecified",
	)
	err := localstackContainer.Container(func() error {
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
	)

	localstackContainer.
		AddQueue("queue1").
		AddQueue("queue2").
		AddQueue("queue3")

	err := localstackContainer.Container(func() error {
		var wg sync.WaitGroup
		wg.Add(75)

		for i := 0; i < 25; i++ {
			go func() {
				defer wg.Done()

				err := localstackContainer.Queues[0].SendMessage(strconv.Itoa(i))
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		for i := 0; i < 25; i++ {
			go func() {
				defer wg.Done()

				err := localstackContainer.Queues[1].SendMessage(strconv.Itoa(i))
				if !assert.NoError(t, err) {
					return
				}
			}()
		}

		for i := 0; i < 25; i++ {
			go func() {
				defer wg.Done()

				err := localstackContainer.Queues[2].SendMessage(strconv.Itoa(i))
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
		easycontainers.ServiceLambda,
	)

	localstackContainer.
		AddFunction("function1", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddFunction("function2", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddFunction("function3", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip")

	err := localstackContainer.Container(func() error {
		var wg sync.WaitGroup
		wg.Add(9)

		for i := 0; i < 3; i++ {
			go func() {
				defer wg.Done()

				err := localstackContainer.Functions[0].SendPayload(map[string]interface{}{
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

				err := localstackContainer.Functions[1].SendPayload(map[string]interface{}{
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

				err := localstackContainer.Functions[2].SendPayload(map[string]interface{}{
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
		easycontainers.ServiceLambda,
	)

	localstackContainer.
		AddFunction("function1", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddFunction("function2", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip").
		AddFunction("function3", "woot", "src/github.com/tsmith-rv/easycontainers/test/handler.zip")

	err := localstackContainer.Container(func() error {
		for _, lambda := range localstackContainer.Functions {
			err := lambda.SendPayload(map[string]interface{}{
				"What is your name?":   3,
				"How old are you?":     33,
				"Some Other Question?": "some answer",
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
