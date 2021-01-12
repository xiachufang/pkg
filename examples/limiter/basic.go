package main

import (
	"fmt"
	"sync"
	"time"

	"github.com/xiachufang/pkg/v2/limiter"
)

func main() {
	// 同一时刻最大只有两个并发
	limit := limiter.New(2)

	var wg sync.WaitGroup
	wg.Add(4)

	for i := 0; i < 4; i++ {
		go func() {
			defer wg.Done()
			if _, ok := limit.Incr(1); ok {
				fmt.Println("run long time task.")
				time.Sleep(time.Second)
			}
		}()
	}

	wg.Wait()
}
