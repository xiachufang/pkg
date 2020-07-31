package storage

import "errors"

var (
	// ErrorCacheMiss 缓存 miss，缓存中不存在该值
	ErrorCacheMiss = errors.New("cache miss")
)
