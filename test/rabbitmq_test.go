package test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/tsmith-rv/easycontainers"
)

func Test_RabbitMQ_Container(t *testing.T) {
	rabbitContainer, port := easycontainers.NewRabbitMQ("Test_RabbitMQ_Container")

	vhost := easycontainers.Vhost{
		Name: "Import",
	}

	exchange := easycontainers.Exchange{
		Name:  "data_exchange",
		Type:  easycontainers.ExchangeTypeDirect,
		Vhost: &vhost,
	}

	queue := easycontainers.Queue{
		Name:    "ha.data_exchange.import",
		Durable: true,
		Vhost:   &vhost,
	}

	binding := easycontainers.QueueBinding{
		Source:      exchange,
		Destination: queue,
		RoutingKey:  "data_import",
		Vhost:       &vhost,
	}

	rabbitContainer.
		AddVhosts(vhost).
		AddExchanges(exchange).
		AddQueue(queue).
		AddBinding(binding)

	err := rabbitContainer.Container(func() error {
		return nil
	})
	if err != nil {
		t.Error(err)
		return
	}

	isFree, err := isPortFree(port)
	if !assert.NoError(t, err) {
		return
	}

	if !assert.True(t, isFree, "port %d should now be available, but isn't", port) {
		return
	}
}
