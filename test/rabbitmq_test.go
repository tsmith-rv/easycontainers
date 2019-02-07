package test

import (
	"testing"

	"github.com/RedVentures/easycontainers"
	"stash.redventures.net/mgpn/callminerimporter/test"
)

func Test_RabbitMQ_Container(t *testing.T) {
	rabbitContainer, _ := test.NewRabbitMQ("some-rabbit-container")

	vhost := test.Vhost{
		Name: "Import",
	}

	exchange := test.Exchange{
		Name:  "data_exchange",
		Type:  easycontainers.ExchangeTypeDirect,
		Vhost: &vhost,
	}

	queue := test.Queue{
		Name:    "ha.data_exchange.import",
		Durable: true,
		Vhost:   &vhost,
	}

	binding := test.QueueBinding{
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
}
