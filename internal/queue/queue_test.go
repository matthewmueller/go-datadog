package queue_test

import (
	"sync"
	"testing"
	"time"

	"github.com/matthewmueller/go-datadog/internal/queue"
)

// concurrency friendly buffer
type buf struct {
	mu sync.Mutex
	b  string
}

func (b *buf) Write(s string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.b += s
}

func (b *buf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.b
}

// func Test
func TestQueue(t *testing.T) {
	var b buf
	q := queue.New(1, 1)
	block := make(chan struct{})

	go func() {
		time.Sleep(200 * time.Millisecond)
		b.Write("2")
		block <- struct{}{}
	}()

	if err := q.Push(func() {
		b.Write("1")
		<-block
	}); err != nil {
		t.Fatal(err)
	}

	if err := q.Push(func() {
		b.Write("3")
	}); err != nil {
		t.Fatal(err)
	}

	q.Wait()
	if b.String() != "123" {
		t.Fatal(b.String() + " != 123")
	}
}

func TestClose(t *testing.T) {
	q := queue.New(1, 1)
	block := make(chan struct{})

	if err := q.Push(func() {
		<-block
	}); err != nil {
		t.Fatal(err)
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		block <- struct{}{}
	}()

	q.Close()

	if err := q.Push(func() {
	}); err == nil {
		t.Fatal("expecting an error")
	} else if err != queue.ErrQueueClosed {
		t.Fatal(err)
	}
}
