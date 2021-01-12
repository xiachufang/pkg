package hacache

import (
	"context"
	"fmt"
	"math/rand"
	"strconv"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/vmihailenco/msgpack/v5"
)

type Value struct {
	value    []byte
	expireAt int64
}

type LocalStorage struct {
	Data map[string]*Value
	mu   sync.Mutex
}

func (s *LocalStorage) Get(key string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if v, ok := s.Data[key]; ok {
		return v.value, nil
	}

	return nil, fmt.Errorf("key %s not found", key)
}

func (s *LocalStorage) Set(key string, value []byte, expiarion time.Duration) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = &Value{value: value, expireAt: time.Now().Unix() + int64(expiarion.Seconds())}

	return nil
}

type MyEncoder struct{}

func (enc *MyEncoder) Encode(v interface{}) ([]byte, error) {
	return msgpack.Marshal(v)
}

func (enc *MyEncoder) Decode(b []byte) (interface{}, error) {
	v := enc.NewValue()
	err := msgpack.Unmarshal(b, v)
	if v, ok := v.(*Foo); ok {
		v.Cached = true
	}

	return v, err
}

func (enc *MyEncoder) NewValue() interface{} {
	return new(Foo)
}

type Foo struct {
	Bar    string
	Cached bool
}

// nolint: unparam
func fn2(bar string) (*Foo, error) {
	return &Foo{Bar: bar}, nil
}

// nolint: errcheck
func TestHaCache_Cache_basic(t *testing.T) {
	hc, err := New(&Options{
		Storage:  &LocalStorage{Data: make(map[string]*Value)},
		GenKeyFn: func(name string) string { return name + "fn2" },
		Fn:       fn2,
		Encoder:  &MyEncoder{},
	})
	if err != nil {
		t.Fatal("init ha-cache error")
	}
	res, err := hc.Do("jack")
	// 这里等待 50ms，让刷新缓存的 goroutine 跑起来
	time.Sleep(50 * time.Millisecond)

	jack, ok := res.(*Foo)
	if !ok || err != nil {
		t.Fatal("decode error: ", err, res)
	}

	if jack.Bar != "jack" || jack.Cached == true {
		t.Fatal("cached value error: ", *jack)
	}

	res2, err := hc.Do("jack")
	jack2, ok := res2.(*Foo)
	if !ok || err != nil {
		t.Fatal("decode error: ", err, res)
	}

	if jack2.Bar != "jack" || jack2.Cached != true {
		t.Fatal("cached value error: ", *jack2)
	}
}

func TestHaCache_SkipCache(t *testing.T) {
	var rd = func(name string) (int, error) { return rand.Int(), nil }
	hc, err := New(&Options{
		Storage:  &LocalStorage{Data: make(map[string]*Value)},
		GenKeyFn: func(name string) string { return SkipCache },
		Fn:       rd,
		Encoder:  &MyEncoder{},
	})

	if err != nil {
		t.Fatal("init ha-cache error")
	}

	v1, err := hc.Do("skip")
	if err != nil {
		t.Fatal("cache run error: ", err)
	}

	v2, err := hc.Do("skip")
	if err != nil {
		t.Fatal("cache run error: ", err)
	}

	if v1.(int) == v2.(int) {
		t.Fatal("skip cache error: ", v1, v2)
	}

	v3, err := hc.Do("skip")
	if err != nil {
		t.Fatal("cache run error: ", err)
	}

	if v3.(int) == v2.(int) {
		t.Fatal("skip cache error: ", v1, v2)
	}
}

type ExpValue struct {
	CreateTS int64
	Value    string
}

func fn3(name string) (*ExpValue, error) {
	return &ExpValue{
		Value:    name,
		CreateTS: time.Now().Unix(),
	}, nil
}

// nolint: errcheck,
func TestHaCache_Cache_expiration(t *testing.T) {
	hc, err := New(&Options{
		Expiration:              time.Second,
		MaxAcceptableExpiration: time.Second * 3,
		Storage:                 &LocalStorage{Data: make(map[string]*Value)},
		GenKeyFn:                func(name string) string { return name + "fn3-exp" },
		Fn:                      fn3,
		Encoder: &HaEncoder{NewValueFn: func() interface{} {
			return new(ExpValue)
		}},
	})
	if err != nil {
		t.Fatal("init hc error: ", err)
	}

	if _, e := hc.Do("aa"); e != nil {
		t.Error(e)
	}

	time.Sleep(2 * time.Second)

	var v *ExpValue
	if result, err := hc.Do("aa"); err == nil {
		v = result.(*ExpValue)
	}

	if v == nil {
		t.Fatal("hit hc cache error")
	}

	// 这里命中了过期缓存，createTS 是一秒前的
	if time.Now().Unix()-v.CreateTS < 1 {
		t.Fatal("hit expired cache error: ", time.Now().Unix(), "/", v.CreateTS)
	}

	// 触发了缓存更新任务，这里拿到的会是最新的
	// 完全过期，强制更新
	time.Sleep(5 * time.Second)
	res, _ := hc.Do("aa")
	now := time.Now().Unix()
	if now-res.(*ExpValue).CreateTS > 1 {
		t.Fatal("background worker error: ", now, res.(*ExpValue).CreateTS)
	}

	hc.opt.MaxAcceptableExpiration = time.Second
	hc.opt.Expiration = time.Second
	// 这里缓存过期时间太长，缓存无效，触发同步更新
	time.Sleep(3 * time.Second)

	res, _ = hc.Do("aa")
	if time.Now().Unix()-res.(*ExpValue).CreateTS > 1 {
		t.Fatal("background worker error")
	}
}

type Fooo struct {
	Value int
}

// nolint: errcheck, unparam
func TestHaCache_Cache_limit(t *testing.T) {
	var maxRun int32 = 2
	var current int32 = 0

	var foo = func(a int) (*Fooo, error) {
		n := atomic.AddInt32(&current, 1)
		if n > maxRun {
			t.Fatal("fn run limiter error, max: ", maxRun, ". now: ", n)
		}
		defer atomic.AddInt32(&current, -1)
		time.Sleep(time.Second)
		return &Fooo{Value: a}, nil
	}

	hc, err := New(&Options{
		Storage:    &LocalStorage{Data: make(map[string]*Value)},
		GenKeyFn:   func(i int) string { return strconv.Itoa(i) + "fn2-limit" },
		Fn:         foo,
		FnRunLimit: 2,
		Encoder: NewEncoder(func() interface{} {
			return Fooo{}
		}),
	})
	if err != nil {
		t.Fatal("init ha-cache error: ", err)
	}

	// 5 个并发请求，缓存同时 miss，导致 5 个请求都会执行 foo，这里 limiter 限制最大只有 2 个执行
	var successCnt int32 = 0
	var wg sync.WaitGroup
	wg.Add(5)
	for i := 0; i < 5; i++ {
		go func(age int) {
			_, err := hc.Do(age)
			if err == nil {
				atomic.AddInt32(&successCnt, 1)
			}
			wg.Done()
		}(i)
	}
	wg.Wait()

	if successCnt != maxRun {
		t.Fatal("expect success: 2, got: ", successCnt)
	}
}

// 测试 context，跳过缓存
func TestHaCache_Context(t *testing.T) {
	var fn = func(ctx context.Context) (int64, error) {
		IgnoreFuncResult(ctx)
		return time.Now().UnixNano(), nil
	}

	hc, err := New(&Options{
		FnRunLimit: 10,
		Storage:    &LocalStorage{Data: make(map[string]*Value)},
		GenKeyFn:   func(ctx context.Context) string { return "TestHaCache_Context" },
		Fn:         fn,
		Expiration: time.Hour,
		Encoder: NewEncoder(func() interface{} {
			return int64(0)
		}),
	})
	if err != nil {
		t.Fatal("init hacache error: ", err)
	}

	v1, _ := hc.Do(context.Background())
	time.Sleep(time.Second)
	v2, _ := hc.Do(context.Background())
	if v1.(int64) == v2.(int64) {
		t.Fatal("cache context test fail: ", v1, v2)
	}

	var fn2 = func() (int64, error) {
		return time.Now().UnixNano(), nil
	}

	hc, err = New(&Options{
		FnRunLimit: 10,
		Storage:    &LocalStorage{Data: make(map[string]*Value)},
		GenKeyFn:   func() string { return "TestHaCache_bbContext" },
		Fn:         fn2,
		Expiration: time.Hour,
		Encoder: NewEncoder(func() interface{} {
			return int64(0)
		}),
	})
	if err != nil {
		t.Fatal("init hacache error: ", err)
	}

	v1, _ = hc.Do()
	time.Sleep(time.Second)
	v2, _ = hc.Do()
	if v1.(int64) != v2.(int64) {
		t.Fatal("cache context test fail: ", v1, v2)
	}
}
