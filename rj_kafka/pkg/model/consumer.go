package model

import "github.com/segmentio/kafka-go"

type ConsuemError struct {
	Message kafka.Message
	Err     error
}
