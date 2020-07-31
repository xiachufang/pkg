package hacache

import "errors"

var (
	// ErrorFnRunLimited 达到原函数执行并发限制
	ErrorFnRunLimited = errors.New("ha-cache fn run rate limited")
	// ErrorInvalidCacheKey 无效的缓存 key
	ErrorInvalidCacheKey = errors.New("invalid cache key")
)
