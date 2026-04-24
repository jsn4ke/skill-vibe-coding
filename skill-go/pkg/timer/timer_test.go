package timer

import (
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestSchedulerAfter(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	var fired atomic.Int32
	s.After(50*time.Millisecond, func() {
		fired.Add(1)
	})

	time.Sleep(100 * time.Millisecond)
	if fired.Load() != 1 {
		t.Fatalf("expected 1 fire, got %d", fired.Load())
	}
}

func TestSchedulerEvery(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	var count atomic.Int32
	s.Every(30*time.Millisecond, func() {
		count.Add(1)
	})

	time.Sleep(120 * time.Millisecond)
	if count.Load() < 2 {
		t.Fatalf("expected at least 2 ticks, got %d", count.Load())
	}
}

func TestSchedulerCancel(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	var fired atomic.Int32
	id := s.After(50*time.Millisecond, func() {
		fired.Add(1)
	})

	s.Cancel(id)
	time.Sleep(100 * time.Millisecond)
	if fired.Load() != 0 {
		t.Fatalf("expected 0 fires after cancel, got %d", fired.Load())
	}
}

func TestSchedulerConcurrentAccess(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	var wg sync.WaitGroup
	var addCount atomic.Int32
	var cancelCount atomic.Int32

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 50; j++ {
				id := s.After(100*time.Millisecond, func() {
					addCount.Add(1)
				})
				if j%3 == 0 {
					s.Cancel(id)
					cancelCount.Add(1)
				}
			}
		}()
	}

	wg.Wait()
	time.Sleep(200 * time.Millisecond)
	t.Logf("added=%d cancelled=%d", addCount.Load(), cancelCount.Load())
}

func TestSchedulerCallbackCanAddTimer(t *testing.T) {
	s := NewScheduler()
	s.Start()
	defer s.Stop()

	var fired atomic.Int32

	s.After(30*time.Millisecond, func() {
		fired.Add(1)
		s.After(30*time.Millisecond, func() {
			fired.Add(1)
		})
	})

	time.Sleep(150 * time.Millisecond)
	if fired.Load() != 2 {
		t.Fatalf("expected 2 fires (chained), got %d", fired.Load())
	}
}

func TestSchedulerStopIdempotent(t *testing.T) {
	s := NewScheduler()
	s.Start()
	time.Sleep(20 * time.Millisecond)

	s.Stop()
	s.Stop()
	s.Stop()
}
