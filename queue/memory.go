package queue

import (
	"context"
	"sync"

	"github.com/vishalss1/raven/types"
)

// Memory is a slice-backed FIFO queue that supports dynamic task insertion.
// Push never blocks. Pop blocks until a task is available, the queue is done,
// or the context is cancelled.
type Memory struct {
	mu    sync.Mutex
	cond  *sync.Cond
	items []types.Task
	done  bool
}

func NewMemory() *Memory {
	q := &Memory{}
	q.cond = sync.NewCond(&q.mu)
	return q
}

// Push appends a task to the queue. Safe to call concurrently.
// Must not be called after Done().
func (q *Memory) Push(task types.Task) {
	q.mu.Lock()
	q.items = append(q.items, task)
	q.mu.Unlock()
	q.cond.Signal()
}

// Pop removes and returns the front task (FIFO). Blocks until a task is
// available. Returns (zero, false) when the queue is done and empty, or
// the context is cancelled.
func (q *Memory) Pop(ctx context.Context) (types.Task, bool) {
	// Watch for context cancellation in a background goroutine.
	done := make(chan struct{})
	defer close(done)
	go func() {
		select {
		case <-ctx.Done():
			q.cond.Broadcast()
		case <-done:
		}
	}()

	q.mu.Lock()
	defer q.mu.Unlock()

	for len(q.items) == 0 && !q.done {
		if ctx.Err() != nil {
			return types.Task{}, false
		}
		q.cond.Wait()
	}

	if ctx.Err() != nil {
		return types.Task{}, false
	}

	if len(q.items) == 0 && q.done {
		return types.Task{}, false
	}

	task := q.items[0]
	q.items = q.items[1:]
	return task, true
}

// Done signals that no more tasks will be pushed. Workers drain remaining
// tasks then exit.
func (q *Memory) Done() {
	q.mu.Lock()
	q.done = true
	q.mu.Unlock()
	q.cond.Broadcast()
}

// Len returns the number of tasks currently buffered.
func (q *Memory) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}
