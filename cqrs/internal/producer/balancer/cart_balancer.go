package balancer

import (
	"strconv"

	"github.com/RoyceAzure/lab/rj_kafka/kafka/producer"
	"github.com/segmentio/kafka-go"
)

type CartBalancer struct {
	BaseBalancer
	producer producer.Producer
}

func NewCartBalancer(producer producer.Producer, numPartitions int) IBaseBalancer {
	return &CartBalancer{BaseBalancer: NewBaseBalancer(numPartitions), producer: producer}
}

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
