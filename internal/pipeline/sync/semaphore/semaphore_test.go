// Copyright 2017 The Go Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package semaphore_test

import (
	"context"
	"math/rand"
	"runtime"
	"sync"
	"testing"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/artefactual-labs/enduro/internal/pipeline/sync/semaphore"
)

const maxSleep = 1 * time.Millisecond

func HammerWeighted(sem *semaphore.Weighted, n int64, loops int) {
	for range loops {
		sem.Acquire(context.Background(), n)
		time.Sleep(time.Duration(rand.Int63n(int64(maxSleep/time.Nanosecond))) * time.Nanosecond) //nolint: gosec
		sem.Release(n)
	}
}

func TestWeighted(t *testing.T) {
	t.Skip()

	t.Parallel()

	n := runtime.GOMAXPROCS(0)
	loops := 10000 / n
	sem := semaphore.NewWeighted(int64(n))
	var wg sync.WaitGroup
	wg.Add(n)
	for i := range n {
		i := i
		go func() {
			defer wg.Done()
			HammerWeighted(sem, int64(i), loops)
		}()
	}
	wg.Wait()
}

func TestWeightedPanic(t *testing.T) {
	t.Parallel()

	defer func() {
		if recover() == nil {
			t.Fatal("release of an unacquired weighted semaphore did not panic")
		}
	}()
	w := semaphore.NewWeighted(1)
	w.Release(1)
}

func TestWeightedTryAcquire(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sem := semaphore.NewWeighted(2)
	tries := []bool{}
	sem.Acquire(ctx, 1)
	tries = append(tries, sem.TryAcquire(1))
	tries = append(tries, sem.TryAcquire(1))

	sem.Release(2)

	tries = append(tries, sem.TryAcquire(1))
	sem.Acquire(ctx, 1)
	tries = append(tries, sem.TryAcquire(1))

	want := []bool{true, false, true, false}
	for i := range tries {
		if tries[i] != want[i] {
			t.Errorf("tries[%d]: got %t, want %t", i, tries[i], want[i])
		}
	}
}

func TestWeightedAcquire(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	sem := semaphore.NewWeighted(2)
	tryAcquire := func(n int64) bool {
		ctx, cancel := context.WithTimeout(ctx, 10*time.Millisecond)
		defer cancel()
		return sem.Acquire(ctx, n) == nil
	}

	tries := []bool{}
	sem.Acquire(ctx, 1)
	tries = append(tries, tryAcquire(1))
	tries = append(tries, tryAcquire(1))

	sem.Release(2)

	tries = append(tries, tryAcquire(1))
	sem.Acquire(ctx, 1)
	tries = append(tries, tryAcquire(1))

	want := []bool{true, false, true, false}
	for i := range tries {
		if tries[i] != want[i] {
			t.Errorf("tries[%d]: got %t, want %t", i, tries[i], want[i])
		}
	}
}

func TestWeightedDoesntBlockIfTooBig(t *testing.T) {
	t.Parallel()

	const n = 2
	sem := semaphore.NewWeighted(n)
	{
		ctx := t.Context()
		go sem.Acquire(ctx, n+1)
	}

	g, ctx := errgroup.WithContext(context.Background())
	for i := n * 3; i > 0; i-- {
		g.Go(func() error {
			err := sem.Acquire(ctx, 1)
			if err == nil {
				time.Sleep(1 * time.Millisecond)
				sem.Release(1)
			}
			return err
		})
	}
	if err := g.Wait(); err != nil {
		t.Errorf("semaphore.NewWeighted(%v) failed to AcquireCtx(_, 1) with AcquireCtx(_, %v) pending", n, n+1)
	}
}

// TestLargeAcquireDoesntStarve times out if a large call to Acquire starves.
// Merely returning from the test function indicates success.
func TestLargeAcquireDoesntStarve(t *testing.T) {
	t.Parallel()

	ctx := context.Background()
	n := int64(runtime.GOMAXPROCS(0))
	sem := semaphore.NewWeighted(n)
	running := true

	var wg sync.WaitGroup
	wg.Add(int(n))
	for i := n; i > 0; i-- {
		sem.Acquire(ctx, 1)
		go func() {
			defer func() {
				sem.Release(1)
				wg.Done()
			}()
			for running {
				time.Sleep(1 * time.Millisecond)
				sem.Release(1)
				sem.Acquire(ctx, 1)
			}
		}()
	}

	sem.Acquire(ctx, n)
	running = false
	sem.Release(n)
	wg.Wait()
}

// translated from https://github.com/zhiqiangxu/util/blob/master/mutex/crwmutex_test.go#L43
func TestAllocCancelDoesntStarve(t *testing.T) {
	sem := semaphore.NewWeighted(10)

	// Block off a portion of the semaphore so that Acquire(_, 10) can eventually succeed.
	sem.Acquire(context.Background(), 1)

	// In the background, Acquire(_, 10).
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		sem.Acquire(ctx, 10)
	}()

	// Wait until the Acquire(_, 10) call blocks.
	for sem.TryAcquire(1) {
		sem.Release(1)
		runtime.Gosched()
	}

	// Now try to grab a read lock, and simultaneously unblock the Acquire(_, 10) call.
	// Both Acquire calls should unblock and return, in either order.
	go cancel()

	err := sem.Acquire(context.Background(), 1)
	if err != nil {
		t.Fatalf("Acquire(_, 1) failed unexpectedly: %v", err)
	}
	sem.Release(1)
}
