package balancer

import (
	"hash/fnv"

	"github.com/segmentio/kafka-go"
)

// ProductIDBalancer 根據 productId 進行分區
type ProductIDBalancer struct {
	numPartitions int
}

func NewProductIDBalancer(numPartitions int) *ProductIDBalancer {
	return &ProductIDBalancer{
		numPartitions: numPartitions,
	}
}

func (b *ProductIDBalancer) Balance(msg kafka.Message, partitions ...int) (partition int) {
	if b.numPartitions <= 0 {
		return 0
	}

	productID := string(msg.Key)
	if productID == "" {
		return 0 // 或者用隨機分配
	}

	hash := fnv.New32a()
	hash.Write([]byte(productID))

	return int(hash.Sum32() % uint32(b.numPartitions))
}
