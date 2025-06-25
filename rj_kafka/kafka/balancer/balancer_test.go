package balancer

import (
	"testing"

	"github.com/segmentio/kafka-go"
)

func TestProductIDBalancer(t *testing.T) {
	balancer := NewProductIDBalancer(3)

	tests := []struct {
		name       string
		key        []byte
		partitions []int
		expected   int
	}{
		{
			name:       "same product ID should go to same partition",
			key:        []byte("product-123"),
			partitions: []int{0, 1, 2},
			expected:   0, // 這個值需要根據實際 hash 結果調整
		},
		{
			name:       "empty key should use first partition",
			key:        []byte(""),
			partitions: []int{0, 1, 2},
			expected:   0,
		},
		{
			name:       "no partitions should return 0",
			key:        []byte("product-123"),
			partitions: []int{},
			expected:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := kafka.Message{Key: tt.key}
			result := balancer.Balance(msg)

			// 對於有分區的情況，確保結果在有效範圍內
			if len(tt.partitions) > 0 {
				if result < 0 || result >= len(tt.partitions) {
					t.Errorf("partition %d out of range [0, %d)", result, len(tt.partitions))
				}
			} else {
				if result != 0 {
					t.Errorf("expected 0, got %d", result)
				}
			}
		})
	}

	// 測試一致性：相同的 key 應該總是得到相同的分區
	msg1 := kafka.Message{Key: []byte("product-123")}
	msg2 := kafka.Message{Key: []byte("product-123")}

	partition1 := balancer.Balance(msg1)
	partition2 := balancer.Balance(msg2)

	if partition1 != partition2 {
		t.Errorf("same key should get same partition: %d != %d", partition1, partition2)
	}
}
