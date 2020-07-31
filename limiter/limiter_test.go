package limiter

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestLimiter(t *testing.T) {
	limiter := New(10)
	if v, ok := limiter.Incr(8); !ok || v != 8 {
		t.Fatal("limit error, expect 8, get ", v)
	}

	limiter.Decr(8)

	v, ok := limiter.Incr(11)
	if v != -1 || ok {
		t.Fatal("limiter check fail")
	}
}

func TestConcurrency(t *testing.T) {
	maxConcurrency := int32(3)
	testConcurrency := 6
	limiter := New(maxConcurrency)

	var wg sync.WaitGroup
	wg.Add(testConcurrency)

	hitTimes := int32(0)

	run := func() {
		atomic.AddInt32(&hitTimes, 1)
		time.Sleep(time.Second)
	}

	for i := 0; i < testConcurrency; i++ {
		go func() {
			defer wg.Done()
			if _, ok := limiter.Incr(1); ok {
				run()
			}
		}()
	}
	wg.Wait()

	if hitTimes != maxConcurrency {
		t.Fatalf("expect concurrent run nums: %d, got: %d\n", maxConcurrency, hitTimes)
	}
}
