package hacache

import (
	"time"
)

// Options ha-cache options struct
type Options struct {
	// 缓存穿透、更新时，执行原函数的最大并发数
	// 达到了这个最大并发数，说明原函数处理时间比较长，
	// 有可能是出现某些故障。
	FnRunLimit int32

	// 最大可接受的过期时间
	// 如果缓存过期，并且过期的时间 > MaxAcceptableExpiration，认为该过期的缓存为可接受的数据，
	// 直接返回并触发异步更新缓存任务。如果 过期时间 < MaxAcceptableExpiration ，则认为改数据
	// 不可用，需要立即更新缓存，并将更新后的数据返回。
	MaxAcceptableExpiration time.Duration

	// 缓存过期时间
	Expiration time.Duration

	// 缓存使用的 storage
	Storage Storage

	// 生成缓存 key 的函数，函数参数必须与 Fn 一致
	GenKeyFn interface{}

	// 被缓存的原函数
	Fn interface{}

	// 事件 channel size
	EventBufferSize int32

	// message encoder
	Encoder Encoder
}

// Storage storage is interface of cache
type Storage interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte, expiration time.Duration) error
}

// Init setup default value of options
func (opt *Options) Init() {
	if opt.EventBufferSize == 0 {
		opt.EventBufferSize = 100
	}

	if opt.EventBufferSize == -1 {
		opt.EventBufferSize = 0
	}

	if opt.FnRunLimit == 0 {
		opt.FnRunLimit = 50
	}

	if opt.MaxAcceptableExpiration == 0 {
		opt.MaxAcceptableExpiration = 10 * time.Minute
	}

	if opt.Expiration == 0 {
		opt.Expiration = 3 * time.Hour
	}

	if opt.Encoder == nil {
		opt.Encoder = &HaEncoder{}
	}
}
