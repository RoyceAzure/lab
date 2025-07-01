package balancer

import (
	"strconv"

	"github.com/segmentio/kafka-go"
)

type CartBalancer struct {
	numPartitions int
}

func NewCartBalancer(numPartitions int) IBaseBalancer {
	return &CartBalancer{numPartitions: numPartitions}
}

// 購物車command 使用userid做key，kafka msgkey必須是userid，所以使用userid做partition
func (c *CartBalancer) Balance(msg kafka.Message, partitions ...int) (partition int) {
	userID, err := strconv.Atoi(string(msg.Key))
	if err != nil {
		return 0
	}

	if len(partitions) != 0 {
		return partitions[userID%len(partitions)]
	}

	return userID % c.numPartitions
}
