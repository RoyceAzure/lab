package balancer

import (
	"github.com/segmentio/kafka-go"
)

type IBaseBalancer interface {
	Balance(msg kafka.Message, partitions ...int) (partition int)
}
