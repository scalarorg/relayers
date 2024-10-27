package btc

import (
	"github.com/scalarorg/relayers/pkg/client/rabbitmq"
)

type Listener struct {
	rmq *rabbitmq.Client
}

func NewListener(rmq *rabbitmq.Client) *Listener {
	return &Listener{rmq: rmq}
}

// Implement BTCListener interface methods
