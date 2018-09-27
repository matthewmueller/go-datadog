package queue

import (
	"errors"
	"sync"
)

// ErrQueueClosed returned when we close the queue
var ErrQueueClosed = errors.New("queue is closed")

// Queue that will never block, if the queue
// is at capacity it will silently drop new
// requests until we have additional capacity.
type Queue struct {
	wg     *sync.WaitGroup
	closed chan struct{}
	jobs   chan func()
}

// New queue
func New(capacity int, concurrency int) *Queue {
	if capacity == 0 {
		panic("capacity must be greater than or equal to 1")
	}
	if concurrency == 0 {
		panic("concurrency must be greater than or equal to 1")
	}

	jobs := make(chan func(), capacity-1)
	closed := make(chan struct{})
	wg := &sync.WaitGroup{}

	// concurrent workers
	for i := 0; i < concurrency; i++ {
		go worker(wg, jobs)
	}

	return &Queue{wg, closed, jobs}
}

func worker(wg *sync.WaitGroup, jobs chan func()) {
	for job := range jobs {
		job()
		wg.Done()
	}
}

// Push a function into the queue be called
// when ready. This will never block but it
// may drop functions if we're at capacity.
func (g *Queue) Push(fn func()) error {
	// handle closed, there may be pushes
	// that are passed this check. Let them
	// be processed
	select {
	case <-g.closed:
		return ErrQueueClosed
	default:
	}

	g.wg.Add(1)
	g.jobs <- fn
	return nil
}

// Wait until the queue has been drained
func (g *Queue) Wait() {
	g.wg.Wait()
}

// Close the queue and wait until it's been drained
func (g *Queue) Close() {
	close(g.closed)
	g.wg.Wait()
}
