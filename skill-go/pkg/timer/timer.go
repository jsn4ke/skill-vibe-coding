package timer

import (
	"sync"
	"sync/atomic"
	"time"
)

// TimerID 是计时器的唯一标识符。
type TimerID uint64

// callback 表示一个定时回调，包含截止时间、间隔和是否重复。
type callback struct {
	id       TimerID
	fn       func()
	deadline time.Time
	interval time.Duration
	repeated bool
}

// Scheduler 是定时调度器，支持一次性定时和周期性定时。
type Scheduler struct {
	nextID   atomic.Uint64
	timers   map[TimerID]*callback
	mu       sync.RWMutex
	running  atomic.Bool
	stopCh   chan struct{}
	stopOnce sync.Once
}

// NewScheduler 创建一个新的调度器。
func NewScheduler() *Scheduler {
	return &Scheduler{
		timers: make(map[TimerID]*callback),
		stopCh: make(chan struct{}),
	}
}

// Start 启动调度器的后台 goroutine。如果已在运行则忽略。
func (s *Scheduler) Start() {
	if s.running.Swap(true) {
		return
	}
	go s.run()
}

// Stop 停止调度器。使用 sync.Once 确保只关闭一次。
func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		s.running.Store(false)
		close(s.stopCh)
	})
}

// After 注册一个一次性定时回调，在指定持续时间后执行。
func (s *Scheduler) After(d time.Duration, fn func()) TimerID {
	id := TimerID(s.nextID.Add(1))
	cb := &callback{
		id:       id,
		fn:       fn,
		deadline: time.Now().Add(d),
	}
	s.mu.Lock()
	s.timers[id] = cb
	s.mu.Unlock()
	return id
}

// Every 注册一个周期性定时回调，按指定间隔重复执行。
func (s *Scheduler) Every(interval time.Duration, fn func()) TimerID {
	id := TimerID(s.nextID.Add(1))
	cb := &callback{
		id:       id,
		fn:       fn,
		deadline: time.Now().Add(interval),
		interval: interval,
		repeated: true,
	}
	s.mu.Lock()
	s.timers[id] = cb
	s.mu.Unlock()
	return id
}

// Cancel 取消指定 ID 的定时回调。
func (s *Scheduler) Cancel(id TimerID) {
	s.mu.Lock()
	delete(s.timers, id)
	s.mu.Unlock()
}

// run 是调度器的主循环，每 10ms 检查一次到期回调。
func (s *Scheduler) run() {
	ticker := time.NewTicker(10 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-s.stopCh:
			return
		case now := <-ticker.C:
			s.tick(now)
		}
	}
}

// tick 检查并执行到期的定时回调。
func (s *Scheduler) tick(now time.Time) {
	var toFire []TimerID

	s.mu.RLock()
	for id, cb := range s.timers {
		if !now.Before(cb.deadline) {
			toFire = append(toFire, id)
		}
	}
	s.mu.RUnlock()

	for _, id := range toFire {
		s.mu.Lock()
		cb, ok := s.timers[id]
		if !ok {
			s.mu.Unlock()
			continue
		}

		isRepeated := cb.repeated
		if isRepeated {
			cb.deadline = now.Add(cb.interval)
		} else {
			delete(s.timers, id)
		}
		fn := cb.fn
		s.mu.Unlock()

		fn()
	}
}
