package util

import (
	"fmt"
	"time"
)

func GenerateOrderID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano()) // 簡化版
}
func GenerateOrderAggregateID(orderID string) string {
	return fmt.Sprintf("order-%s", orderID)
}
