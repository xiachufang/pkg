package hacache

import (
	"sync/atomic"
	"time"

	"github.com/smira/go-statsd"
)

const defaultExportInterval = 5 * time.Second

// MetricType 指标类型
type MetricType string

// GaugeMetricType gauge 指标类型
type GaugeMetricType string

const (
	// MHit 命中有效缓存
	MHit MetricType = "hit"
	// MMiss 完全 miss
	MMiss MetricType = "miss"
	// MMissExpired 命中过期缓存，但是在可接受过期范围内
	MMissExpired MetricType = "miss-expired"
	// MMissInvalid 命中过期缓存，在最大可接受失效时间范围外
	MMissInvalid MetricType = "miss-invalid"
	// MInvalidReturned 强制返回过期缓存
	MInvalidReturned MetricType = "invalid-returned"
	// MFnRun 执行原函数
	MFnRun MetricType = "fn-run"
	// MFnRunErr 原函数执行出错
	MFnRunErr MetricType = "fn-run-err"
	// MFnRunLimited 原函数执行被限流
	MFnRunLimited MetricType = "fn-run-limited"
	// MEventChanBlocked 事件 channel block 住
	MEventChanBlocked MetricType = "event-chan-blocked"
	// MSkip 不缓存
	MSkip MetricType = "skip"
	// MWorkerPanic worker goroutine panic times
	MWorkerPanic MetricType = "worker-panic"
	// GMFnRunConcurrency 原函数执行并发度
	GMFnRunConcurrency GaugeMetricType = "fn-run-concurrency"
)

// Stats 缓存统计数据
type Stats struct {
	// 命中有效缓存次数
	Hit int32
	// 命中失效缓存次数
	MissExpired int32
	// 命中在最大可接受失效时间范围外的次数
	MissInvalid int32
	// 强制返回过期缓存次数
	InvalidReturned int32
	// 完全 miss
	Miss int32
	// 原函数执行次数
	FnRun int32
	// 原函数执行被限流
	FnRunLimited     int32
	FnRunErr         int32
	EventChanBlocked int32
	Skip             int32
	WorkerPanic      int32

	// FnRun 当前执行并发度
	FnRunConcurrency int32

	Exporter *statsd.Client
}

// Gauge 设置某项指标 gauge 数据
func (s *Stats) Gauge(m GaugeMetricType, i int32) {
	if m == GMFnRunConcurrency {
		atomic.StoreInt32(&s.FnRunConcurrency, i)
	}
}

// Incr 增加某项指标数据
func (s *Stats) Incr(m MetricType, i int32) {
	switch m {
	case MHit:
		atomic.AddInt32(&s.Hit, i)
	case MMiss:
		atomic.AddInt32(&s.Miss, i)
	case MMissExpired:
		atomic.AddInt32(&s.MissExpired, i)
	case MMissInvalid:
		atomic.AddInt32(&s.MissInvalid, i)
	case MFnRun:
		atomic.AddInt32(&s.FnRun, i)
	case MInvalidReturned:
		atomic.AddInt32(&s.InvalidReturned, i)
	case MFnRunErr:
		atomic.AddInt32(&s.FnRunErr, i)
	case MFnRunLimited:
		atomic.AddInt32(&s.FnRunLimited, i)
	case MEventChanBlocked:
		atomic.AddInt32(&s.EventChanBlocked, i)
	case MSkip:
		atomic.AddInt32(&s.Skip, i)
	case MWorkerPanic:
		atomic.AddInt32(&s.WorkerPanic, i)
	}
}

// Export 到处统计数据，并清空
func (s *Stats) Export() map[MetricType]int32 {
	return map[MetricType]int32{
		MHit:              atomic.SwapInt32(&s.Hit, 0),
		MMissExpired:      atomic.SwapInt32(&s.MissExpired, 0),
		MMissInvalid:      atomic.SwapInt32(&s.MissInvalid, 0),
		MMiss:             atomic.SwapInt32(&s.Miss, 0),
		MFnRun:            atomic.SwapInt32(&s.FnRun, 0),
		MInvalidReturned:  atomic.SwapInt32(&s.InvalidReturned, 0),
		MFnRunLimited:     atomic.SwapInt32(&s.FnRunLimited, 0),
		MFnRunErr:         atomic.SwapInt32(&s.FnRunErr, 0),
		MEventChanBlocked: atomic.SwapInt32(&s.EventChanBlocked, 0),
		MSkip:             atomic.SwapInt32(&s.Skip, 0),
		MWorkerPanic:      atomic.SwapInt32(&s.WorkerPanic, 0),
	}
}

// ExportGauge 获取 Gauge 数据
func (s *Stats) ExportGauge() map[GaugeMetricType]int32 {
	return map[GaugeMetricType]int32{
		GMFnRunConcurrency: atomic.LoadInt32(&s.FnRunConcurrency),
	}
}

// Run 上报数据
func (s *Stats) Run() {
	defer func() {
		if v := recover(); v != nil {
			s.Exporter.Incr("panic", 1, statsd.StringTag("tag", "hacache-stats"))
			s.Run()
		}
	}()

	for {
		time.Sleep(defaultExportInterval)
		data := s.Export()

		if s.Exporter == nil {
			continue
		}

		for metric, value := range data {
			if value == 0 {
				continue
			}
			s.Exporter.Incr("ha-cache", int64(value), statsd.StringTag("m", string(metric)))
		}

		for gaugeMetrics, value := range s.ExportGauge() {
			s.Exporter.Gauge("ha-cache-gauge", int64(value), statsd.StringTag("m", string(gaugeMetrics)))
		}
	}
}

// Setup 设置 statsd 连接
func (s *Stats) Setup(client *statsd.Client) {
	if s.Exporter != nil || client == nil {
		return
	}

	s.Exporter = client

	go s.Run()
}

// CurrentStats 全局统计实例
var CurrentStats = new(Stats)
