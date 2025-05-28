package error

import "fmt"

type RedisErrorCode int

const (
	RedisErrorUnknown RedisErrorCode = iota
	RedisErrorConnection
	RedisErrorTimeout
	RedisErrorInvalid
)

type RedisError struct {
	Code    RedisErrorCode `json:"code"`
	Message string         `json:"message"`
}

func (e *RedisError) Error() string {
	return fmt.Sprintf("code: %d, message: %s", e.Code, e.Message)
}
