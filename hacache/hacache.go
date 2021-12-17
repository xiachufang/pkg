package hacache

import (
	"errors"
	"fmt"
	"reflect"
	"runtime/debug"
	"time"

	"github.com/vmihailenco/msgpack/v5"
	"github.com/xiachufang/pkg/v2/hacache/storage"
	"github.com/xiachufang/pkg/v2/limiter"
	"go.uber.org/zap"
)

// SkipCache 当缓存 key 为 SkipCache 值时，跳过缓存
const SkipCache = "__hacache_skip_cache__"

// Event 拉取缓存时，触发的事件类型
type Event interface{}

// EventCacheExpired 缓存过期，但是可以接受，需要执行原始函数进行更新
type EventCacheExpired struct {
	Args []interface{}
}

// EventCacheInvalid 缓存无效，需要立即更新
type EventCacheInvalid struct {
	// Bytes 原始函数返回的结果，需要放到缓存里
	Data interface{}
	// Key 缓存 key
	Key string
}

// HaCache ha-cache struct
type HaCache struct {
	// fnRunLimiter 被缓存的原函数执行并发限制
	fnRunLimiter *limiter.Limiter
	opt          *Options
	events       chan Event
	logger       *zap.Logger
}

// CachedValue 缓存值类型
type CachedValue struct {
	// protobuf message 序列化之后的 bytes
	Bytes []byte
	// 缓存创建的时间戳/s
	CreateTS int64
}

// FnResult 被缓存函数返回值的通用结构
type FnResult struct {
	// Val 原函数返回值
	Val interface{}
	// Err 原函数返回的 error
	Err error
	// Ignore 忽略返回值，不设置回缓存
	Ignore bool
}

// New return a new ha-cache instance
// nolint: gomnd
func New(opt *Options) (*HaCache, error) {
	if opt.Storage == nil {
		return nil, errors.New("no storage found")
	}
	opt.Init()

	if reflect.ValueOf(opt.Fn).Type().NumOut() != 1 {
		return nil, errors.New("fn return value must be `*hacache.FnResult`")
	}
	returnType := reflect.TypeOf(opt.Fn).Out(0)
	if returnType != reflect.TypeOf((*FnResult)(nil)) {
		return nil, errors.New("fn return value must be `*hacache.FnResult`")
	}

	hc := &HaCache{
		fnRunLimiter: limiter.New(opt.FnRunLimit),
		opt:          opt,
		events:       make(chan Event, opt.EventBufferSize),
		logger:       opt.Logger,
	}
	go hc.worker()
	return hc, nil
}

// worker 刷新缓存、更新过期缓存
func (hc *HaCache) worker() {
	defer func() {
		if v := recover(); v != nil {
			CurrentStats.Incr(MWorkerPanic, 1)
			hc.logger.Error(fmt.Sprintf("hacache worker paniced: %v, stack: %s", v, string(debug.Stack())))
			hc.worker()
		}
	}()

	for {
		event := <-hc.events
		switch e := event.(type) {
		case *EventCacheExpired:
			data, err := hc.FnRun(true, e.Args...)
			if err != nil || (data != nil && (data.Err != nil || data.Ignore)) {
				continue
			}
			if err := hc.Set(hc.GenCacheKey(e.Args...), data.Val); err != nil {
				continue
			}
		case *EventCacheInvalid:
			if err := hc.Set(e.Key, e.Data); err != nil {
				continue
			}
		}
	}
}

// FnRun 执行原函数，原函数执行时，受并发限制，
// 如果是缓存过期异步更新，触发限流直接跳过；
// 如果是缓存失效同步更新，触发限流服务报错
// 被缓存的函数签名为: func(args ...interface{}) (*FnResult)
func (hc *HaCache) FnRun(background bool, args ...interface{}) (*FnResult, error) {
	CurrentStats.Incr(MFnRun, 1)
	_, ok := hc.fnRunLimiter.Incr(1)
	defer hc.fnRunLimiter.Decr(1)

	// 统计当前原函数执行的并发度
	CurrentStats.Gauge(GMFnRunConcurrency, hc.fnRunLimiter.GetCurrent())

	if !ok {
		CurrentStats.Incr(MFnRunLimited, 1)
	}

	// 异步更新的直接跳过，需要同步更新的返回报错
	if !ok && background {
		return nil, nil
	} else if !ok && !background {
		return nil, ErrorFnRunLimited
	}

	result, err := call(hc.opt.Fn, args...)
	if err != nil {
		return nil, err
	}

	v, ok := result[0].Interface().(*FnResult)
	if ok {
		return v, nil
	}

	return nil, fmt.Errorf("fnResult type convert error")
}

// GenCacheKey 生成缓存 key
func (hc *HaCache) GenCacheKey(args ...interface{}) string {
	result, err := call(hc.opt.GenKeyFn, args...)
	if err != nil {
		return ""
	}

	// 生成缓存 key 的函数只有一个 string 返回值
	if len(result) != 1 {
		return ""
	}

	if key, ok := result[0].Interface().(string); ok {
		return key
	}

	return ""
}

// Get get cached value
func (hc *HaCache) Get(key string) (*CachedValue, error) {
	b, err := hc.opt.Storage.Get(key)
	if err != nil {
		return nil, err
	}

	v := new(CachedValue)
	err = msgpack.Unmarshal(b, v)
	return v, err
}

// Set set `key` to `msg`
func (hc *HaCache) Set(key string, data interface{}) error {
	// protobuf message 用 protobuf 序列化
	// 带上 create time 时间戳的 struct 用 msgpack 序列化
	b, err := hc.opt.Encoder.Encode(data)
	if err != nil {
		return err
	}

	value, err := msgpack.Marshal(CachedValue{
		Bytes:    b,
		CreateTS: time.Now().Unix(),
	})
	if err != nil {
		return err
	}
	return hc.opt.Storage.Set(key, value, 0)
}

// Trigger 触发某个 event (non-blocking)
func (hc *HaCache) Trigger(event Event) {
	select {
	case hc.events <- event:
		return
	default:
		CurrentStats.Incr(MEventChanBlocked, 1)
	}
}

// Do 取缓存结果，如果不存在，则更新缓存
func (hc *HaCache) Do(args ...interface{}) (interface{}, error) {
	cacheKey := hc.GenCacheKey(args...)
	if cacheKey == "" {
		return nil, ErrorInvalidCacheKey
	} else if cacheKey == SkipCache {
		CurrentStats.Incr(MSkip, 1)
		res, err := hc.FnRun(false, args...)
		if err != nil {
			return nil, err
		}
		return res.Val, res.Err
	}

	value, err := hc.Get(cacheKey)
	// 这里取缓存出错，一般可认为是没取到缓存，极端情况可能是 Redis 异常，直接穿透到原函数返回，并刷新缓存
	// 原函数执行受 FnRunLimiter 并发限制
	if err == storage.ErrorCacheMiss {
		CurrentStats.Incr(MMiss, 1)
	}

	// 缓存 miss，执行原函数
	if err != nil {
		res, err := hc.FnRun(false, args...)
		if err != nil || res.Err != nil {
			CurrentStats.Incr(MFnRunErr, 1)
			return nil, err
		}

		if !res.Ignore {
			hc.Trigger(&EventCacheInvalid{
				Data: res.Val,
				Key:  cacheKey,
			})
		}
		return res.Val, nil
	}

	expireAt := value.CreateTS + int64(hc.opt.Expiration.Seconds())
	now := time.Now().Unix()

	// 缓存值在有效期内
	if expireAt >= now {
		CurrentStats.Incr(MHit, 1)
		return hc.opt.Encoder.Decode(value.Bytes)
	}

	// 缓存过期已经超过了最大可接受时间，需要同步更新缓存，并返回最新内容
	if now > (expireAt + int64(hc.opt.MaxAcceptableExpiration.Seconds())) {
		CurrentStats.Incr(MMissInvalid, 1)
		res, err := hc.FnRun(false, args...)
		// 触发限流、或者原函数执行错误，强制返回过期数据，并且跳过缓存更新步骤
		if err != nil || res.Err != nil {
			CurrentStats.Incr(MInvalidReturned, 1)
			return hc.opt.Encoder.Decode(value.Bytes)
		}

		if !res.Ignore {
			hc.Trigger(&EventCacheInvalid{
				Data: copyVal(res.Val),
				Key:  cacheKey,
			})
		}

		return res.Val, nil
	}

	CurrentStats.Incr(MMissExpired, 1)
	// 缓存过期，但是在可接受的过期范围内，返回缓存内容，并触发更新任务
	v, err := hc.opt.Encoder.Decode(value.Bytes)
	if err == nil {
		hc.Trigger(&EventCacheExpired{
			Args: args,
		})
	}
	return v, err
}
