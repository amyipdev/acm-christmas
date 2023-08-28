package concqueue

import (
	"context"
	"strconv"
	"testing"

	"github.com/alecthomas/assert/v2"
)

func TestQueue(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	q := NewQueue(func(_ context.Context, v int) string {
		return strconv.Itoa(v)
	})
	go func() { q.Run(ctx) }()

	go func() {
		q.Enqueue(ctx, 1)
		q.Enqueue(ctx, 2)
		q.Enqueue(ctx, 3)
	}()

	items, err := q.DequeueN(ctx, 3)
	assert.NoError(t, err)
	assert.Equal(t, []string{"1", "2", "3"}, items)
}

func BenchmarkQueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	b.Cleanup(cancel)

	q := NewQueue(func(_ context.Context, v int) int { return v })
	go func() { q.Run(ctx) }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(ctx, i)
		q.Dequeue(ctx)
	}
}
