package limiter

import "sync/atomic"

// Limiter 并发限制
type Limiter struct {
	Current        int32
	MaxConcurrency int32
}

// New create a concurrency limiter with MaxConcurrency=maxConcurrency
func New(maxConcurrency int32) *Limiter {
	return &Limiter{
		MaxConcurrency: maxConcurrency,
	}
}

// Incr increase current concurrency, if (-1, false) returned, reject request.
func (limiter *Limiter) Incr(v int32) (int32, bool) {
	if limiter.MaxConcurrency <= 0 {
		return 0, true
	}

	current := atomic.AddInt32(&limiter.Current, v)
	if current > limiter.MaxConcurrency {
		return -1, false
	}

	return current, true
}

// Decr decrease/release current concurrency.
func (limiter *Limiter) Decr(v int32) (int32, bool) {
	if limiter.MaxConcurrency <= 0 {
		return 0, true
	}

	return atomic.AddInt32(&limiter.Current, -v), true
}

// GetCurrent get current concurrency
func (limiter *Limiter) GetCurrent() int32 {
	return atomic.LoadInt32(&limiter.Current)
}
