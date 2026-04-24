package timer

import (
	"sync"
	"sync/atomic"
	"time"
)

type TimerID uint64

type callback struct {
	id       TimerID
	fn       func()
	deadline time.Time
	interval time.Duration
	repeated bool
}

type Scheduler struct {
	nextID   atomic.Uint64
	timers   map[TimerID]*callback
	mu       sync.RWMutex
	running  atomic.Bool
	stopCh   chan struct{}
	stopOnce sync.Once
}

func NewScheduler() *Scheduler {
	return &Scheduler{
		timers: make(map[TimerID]*callback),
		stopCh: make(chan struct{}),
	}
}

func (s *Scheduler) Start() {
	if s.running.Swap(true) {
		return
	}
	go s.run()
}

func (s *Scheduler) Stop() {
	s.stopOnce.Do(func() {
		s.running.Store(false)
		close(s.stopCh)
	})
}

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

func (s *Scheduler) Cancel(id TimerID) {
	s.mu.Lock()
	delete(s.timers, id)
	s.mu.Unlock()
}

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
