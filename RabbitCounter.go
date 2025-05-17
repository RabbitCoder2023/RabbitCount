package RabbitCount

import (
	"strconv"
	"sync"
	"sync/atomic"
	"time"
)

type RabbitCount struct {
	counter    Counter
	interval   time.Duration
	running    int32
	onStop     func(r *RabbitCount)
	onStopLock sync.RWMutex

	maxCPM     int64
	maxCPMLock sync.Mutex
}

func NewRabbitCount(interval time.Duration) *RabbitCount {
	return &RabbitCount{
		interval: interval,
	}
}

func (r *RabbitCount) OnStop(f func(*RabbitCount)) {
	r.onStopLock.Lock()
	r.onStop = f
	r.onStopLock.Unlock()
}

func (r *RabbitCount) run() {
	if !atomic.CompareAndSwapInt32(&r.running, 0, 1) {
		return
	}

	go func() {
		ticker := time.NewTicker(r.interval)
		defer ticker.Stop()

		for {
			<-ticker.C
			r.counter.Reset()
			atomic.StoreInt32(&r.running, 0)

			r.onStopLock.RLock()
			if r.onStop != nil {
				r.onStop(r)
			}
			r.onStopLock.RUnlock()
			return
		}
	}()
}

func (r *RabbitCount) Incr(val int64) {
	r.counter.Incr(val)

	// actualizar maxCPM si es necesario
	r.maxCPMLock.Lock()
	if r.counter.Value() > r.maxCPM {
		r.maxCPM = r.counter.Value()
	}
	r.maxCPMLock.Unlock()

	r.run()
}

func (r *RabbitCount) Rate() int64 {
	return r.counter.Value()
}

func (r *RabbitCount) MaxCPM() int64 {
	r.maxCPMLock.Lock()
	defer r.maxCPMLock.Unlock()
	return r.maxCPM
}

func (r *RabbitCount) String() string {
	return strconv.FormatInt(r.counter.Value(), 10)
}

// --- Counter

type Counter int64

func (c *Counter) Incr(val int64) {
	atomic.AddInt64((*int64)(c), val)
}

func (c *Counter) Reset() {
	atomic.StoreInt64((*int64)(c), 0)
}

func (c *Counter) Value() int64 {
	return atomic.LoadInt64((*int64)(c))
}
