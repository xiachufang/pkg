package hacache

import (
	"context"
	"sync"
	"sync/atomic"
)

const (
	cacheResult     = 1
	ignoreResult    = -1
	cacheContextKey = "cacheContext"
)

type contextKey string

// CacheContext 缓存的一些上下文信息
type CacheContext struct {
	sync.Mutex

	// cacheResult 是否需要缓存原函数返回结果
	// v < 0 表示不缓存， >= 0 表示缓存
	cacheResult int64
}

// IgnoreResult 不缓存原函数返回结果
func (ctx *CacheContext) IgnoreResult() {
	atomic.StoreInt64(&ctx.cacheResult, ignoreResult)
}

// CacheResult 是否缓存结果
func (ctx *CacheContext) CacheResult() bool {
	v := atomic.LoadInt64(&ctx.cacheResult)
	return v == cacheResult
}

// IgnoreFuncResult 不缓存原函数返回结果
func IgnoreFuncResult(ctx context.Context) {
	cacheCtx, ok := ctx.Value(contextKey(cacheContextKey)).(*CacheContext)
	if !ok {
		return
	}

	cacheCtx.IgnoreResult()
}

// CacheResult 是否缓存结果, default is true
func CacheResult(ctx context.Context) bool {
	cacheCtx, ok := ctx.Value(contextKey(cacheContextKey)).(*CacheContext)
	if !ok {
		return true
	}

	return cacheCtx.CacheResult()
}

// WrapCacheContext wrap new cache context
func WrapCacheContext(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKey(cacheContextKey), &CacheContext{cacheResult: cacheResult})
}
