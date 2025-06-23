package balancer

import (
	"github.com/segmentio/kafka-go"
)

type BaseBalancer struct {
	numPartitions int
}

type IBaseBalancer interface {
	Balance(msg kafka.Message, partitions ...int) (partition int)
}

func NewBaseBalancer(numPartitions int) BaseBalancer {
	return BaseBalancer{numPartitions: numPartitions}
}
