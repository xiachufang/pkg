# pkg

![Test](https://github.com/xiachufang/pkg/workflows/Test/badge.svg) ![Lint](https://github.com/xiachufang/pkg/workflows/Lint/badge.svg)

Common Go utilities used by Xiachufang Engineering Team.


## Packages


### [`limiter`](https://github.com/xiachufang/pkg/tree/master/limiter)

Concurrency limiter.

```go
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
```


### [`logger`](https://github.com/xiachufang/pkg/tree/master/logger)

Syslog helper.

```go
func main() {
	l, err := logger.New(&logger.Options{
		Tag:   "examples.logger",
		Debug: true,
	})

	if err != nil {
		panic(err)
	}

	l.Debug("debug message")
	l.Info("info message")
	l.Warn("info message")
	l.Error("info message")
}
```

### [`hacache`](https://github.com/xiachufang/pkg/tree/master/hacache)

Cache component.

```go
func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr:               "localhost:6379",
	})

	cache, err := hacache.New(&hacache.Options{
		FnRunLimit:              10,	// LongTimeTask 同一时刻最多允许 10 个并发穿透到被缓存的原函数
		MaxAcceptableExpiration: 10 * time.Minute,	// 如果命中的缓存过期时间在 10 分钟内，返回缓存值，并异步更新缓存值
		Expiration:              time.Hour,	// 过期时间 1 小时
		Storage:                 storage.NewRedis(redisClient),
		GenKeyFn:                GenerateCacheKey,	// 缓存 key 生成函数，函数参数必须与 Fn 一致
		Fn:                      LongTimeTask,	// 被缓存的原函数
		Encoder:                 hacache.NewEncoder(func() interface{} {
			return ""	// 缓存值为 string 类型，返回缓存值类型空值
		}),
	})

	if err != nil {
		panic(err)
	}

	tom, err := cache.Do("tom", 10)
	if err != nil {
		panic(err)
	}

	// me == "tom is 10 years old"
	fmt.Println(tom.(string))

	tom2, err := cache.Do("tom", 20)
	if err != nil {
		panic(err)
	}

	// tom2 == "tom is 10 years old"
	fmt.Println(tom2.(string))
}
```

