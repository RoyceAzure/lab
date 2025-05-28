package redis_client

import (
	"sync"

	"github.com/redis/go-redis/v9"
)

var (
	_instances = sync.Map{}
)

func GetRedisClient(address string, options ...Option) (*redis.Client, error) {
	client, ok := _instances.Load(address)
	if !ok {
		var err error
		client, err = createRedisClient(address, options...)
		if err != nil {
			return nil, err
		}
		_instances.Store(address, client)
	}

	return client.(*redis.Client), nil
}

func createRedisClient(address string, options ...Option) (*redis.Client, error) {
	opts := &redis.Options{
		Addr: address,
	}

	for _, option := range options {
		option(opts)
	}

	return redis.NewClient(opts), nil
}

type Option func(*redis.Options)

func WithPassword(password string) Option {
	return func(o *redis.Options) {
		o.Password = password
	}
}

func WithDB(db int) Option {
	return func(o *redis.Options) {
		o.DB = db
	}
}

func WithPoolSize(poolSize int) Option {
	return func(o *redis.Options) {
		o.PoolSize = poolSize
	}
}
