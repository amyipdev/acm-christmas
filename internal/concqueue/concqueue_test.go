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

	q := NewQueue(func(_ context.Context, v int) (string, error) {
		return strconv.Itoa(v), nil
	})
	go func() { q.Run(ctx) }()

	go func() {
		q.Enqueue(ctx, 1)
		q.Enqueue(ctx, 2)
		q.Enqueue(ctx, 3)
	}()

	items, err := q.DequeueN(ctx, 3)
	assert.NoError(t, err)
	assert.Equal(t, []Result[string]{Ok("1"), Ok("2"), Ok("3")}, items)
}

func BenchmarkQueue(b *testing.B) {
	ctx, cancel := context.WithCancel(context.Background())
	b.Cleanup(cancel)

	q := NewQueue(func(_ context.Context, v int) (int, error) { return v, nil })
	go func() { q.Run(ctx) }()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		q.Enqueue(ctx, i)
		q.Dequeue(ctx)
	}
}
