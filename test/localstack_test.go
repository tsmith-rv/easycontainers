package test

import (
	"testing"

	"github.com/RedVentures/easycontainers"
)

func Test_Localstack_Container(t *testing.T) {
	localstackContainer, _ := easycontainers.NewLocalstack(
		"localstack-Test_Localstack_Container",
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

	localstackContainer.AddSQSQueue("somequeue", []string{"msg1", "msg2", "msg3"})

	err := localstackContainer.Container(func() error {
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}
}
