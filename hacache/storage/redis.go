package storage

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// Redis redis storage
type Redis struct {
	client *redis.Client
}

var (
	// ErrNilRedis not init redis client
	ErrNilRedis = errors.New("redis not found")
)

// Get redis GET
func (r *Redis) Get(key string) ([]byte, error) {
	if r.client == nil {
		return nil, ErrNilRedis
	}

	v := r.client.Get(context.Background(), key)
	if v.Err() != nil {
		if strings.Contains(v.Err().Error(), "redis: nil") {
			return nil, ErrorCacheMiss
		}

		return nil, v.Err()
	}

	return v.Bytes()
}

// Set redis SET
func (r *Redis) Set(key string, value []byte, expiration time.Duration) error {
	if r.client == nil {
		return ErrNilRedis
	}

	v := r.client.Set(context.Background(), key, value, expiration)
	return v.Err()
}

// NewRedis return a new redis storage
func NewRedis(client *redis.Client) *Redis {
	return &Redis{client: client}
}
