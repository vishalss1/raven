package queue

import (
	"context"
	"sync"

	"github.com/vishalss1/raven/types"
)

// Memory is a simple buffered-channel Queue.
// Araneae calls Push to enqueue Tasks; the Engine calls Pop to dequeue them.
// Call Close when no more tasks will be pushed — workers drain the buffer
// then stop.
type Memory struct {
	ch   chan types.Task
	once sync.Once
}

func NewMemory(bufferSize int) *Memory {
	if bufferSize <= 0 {
		bufferSize = 512
	}
	return &Memory{ch: make(chan types.Task, bufferSize)}
}

// Push enqueues a Task. Blocks if the buffer is full.
// Must not be called after Close.
func (q *Memory) Push(task types.Task) {
	q.ch <- task
}

// Pop dequeues the next Task. Returns (task, true) on success.
// Returns (zero, false) if the queue is closed and drained, or ctx is done.
func (q *Memory) Pop(ctx context.Context) (types.Task, bool) {
	select {
	case task, ok := <-q.ch:
		return task, ok
	case <-ctx.Done():
		return types.Task{}, false
	}
}

// Close signals that no more tasks will be pushed.
// Workers drain remaining tasks then exit.
func (q *Memory) Close() {
	q.once.Do(func() { close(q.ch) })
}

// Len returns the number of tasks currently buffered (approximate).
func (q *Memory) Len() int {
	return len(q.ch)
}
