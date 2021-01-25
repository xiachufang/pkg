package main

import (
	"context"
	"fmt"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/smira/go-statsd"
	"github.com/xiachufang/pkg/v2/hacache"
	"github.com/xiachufang/pkg/v2/hacache/storage"
)

// GenerateCacheKey generate cache key
func GenerateCacheKey(name string, age int) string {
	return name
}

// LongTimeTask cached func.
func LongTimeTask(name string, age int) *hacache.FnResult {
	time.Sleep(time.Second)
	return &hacache.FnResult{
		Val: fmt.Sprintf("%s is %d years old.\n", name, age),
		Err: nil,
	}
}

func main() {
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})
	statsdClient := statsd.NewClient("localhost:8125")

	// 初始化监控上报连接
	hacache.CurrentStats.Setup(statsdClient)

	cache, err := hacache.New(&hacache.Options{
		FnRunLimit:              10,               // LongTimeTask 同一时刻最多允许 10 个并发穿透到被缓存的原函数
		MaxAcceptableExpiration: 10 * time.Minute, // 如果命中的缓存过期时间在 10 分钟内，返回缓存值，并异步更新缓存值
		Expiration:              time.Hour,        // 过期时间 1 小时
		Storage:                 storage.NewRedis(redisClient),
		GenKeyFn:                GenerateCacheKey, // 缓存 key 生成函数，函数参数必须与 Fn 一致
		Fn:                      LongTimeTask,     // 被缓存的原函数
		Encoder: hacache.NewEncoder(func() interface{} {
			return "" // 缓存值为 string 类型，返回缓存值类型空值
		}),
	})

	if err != nil {
		panic(err)
	}

	tom, err := cache.Do(context.Background(), "tom", 10)
	if err != nil {
		panic(err)
	}

	// tom == "tom is 10 years old"
	fmt.Println(tom.(string))

	tom2, err := cache.Do(context.Background(), "tom", 20)
	if err != nil {
		panic(err)
	}

	// tom2 == "tom is 10 years old"
	fmt.Println(tom2.(string))
}
