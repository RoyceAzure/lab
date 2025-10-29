package model

import (
	"time"

	"github.com/segmentio/kafka-go"
)

// Header 代表 Kafka 消息的標頭
// 可用於存儲消息的元數據信息，如版本、壓縮方式、追蹤ID等
type Header struct {
	// Key 標頭的鍵名
	Key string
	// Value 標頭的值，使用 []byte 以支持各種數據類型
	Value []byte
}

// Message 代表一個 Kafka 消息
// 在 Kafka 中，消息是最基本的數據單位，包含了數據本身和相關的元數據
type Message struct {
	// Key 是消息的鍵
	// 用於分區分配，相同的 Key 會被分配到相同的分區
	// 常用於保證相關消息的順序性
	Key []byte

	// Value 是消息的實際內容
	// 可以是任何格式的數據：JSON、Protobuf、Avro 等
	Value []byte

	// Topic 是消息所屬的主題
	// Kafka 使用主題來組織和分類消息
	// 生產者發送到特定主題，消費者從特定主題訂閱
	Topic string

	// Partition 是消息所在的分區號
	// Kafka 使用分區來實現並行處理和擴展性
	// 每個主題可以有多個分區，每個分區都是一個有序的消息序列
	Partition int

	// Offset 是消息在分區中的位置
	// 每個消息在分區中都有一個唯一的遞增偏移量
	// 用於追踪消費進度和消息定位
	Offset int64

	// Headers 是消息的標頭列表
	// 可以包含多個鍵值對，用於存儲消息相關的元數據
	// 例如：追蹤ID、消息類型、版本號等
	Headers []Header

	// Time 是消息的時間戳
	// 可以是消息產生時間或寫入時間
	// 用於時間基準和消息追蹤
	Time time.Time
}

// ToKafkaMessage converts our Message to kafka-go Message
func (m *Message) ToKafkaMessage() kafka.Message {
	headers := make([]kafka.Header, len(m.Headers))
	for i, h := range m.Headers {
		headers[i] = kafka.Header{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	return kafka.Message{
		Key:       m.Key,
		Value:     m.Value,
		Topic:     m.Topic,
		Partition: m.Partition,
		Offset:    m.Offset,
		Headers:   headers,
		Time:      m.Time,
	}
}

// FromKafkaMessage converts kafka-go Message to our Message
func FromKafkaMessage(km kafka.Message) Message {
	headers := make([]Header, len(km.Headers))
	for i, h := range km.Headers {
		headers[i] = Header{
			Key:   h.Key,
			Value: h.Value,
		}
	}

	return Message{
		Key:       km.Key,
		Value:     km.Value,
		Topic:     km.Topic,
		Partition: km.Partition,
		Offset:    km.Offset,
		Headers:   headers,
		Time:      km.Time,
	}
}
