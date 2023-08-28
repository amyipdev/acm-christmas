package concqueue

import (
	"context"
	"errors"
	"sync/atomic"
)

// Queue implements a concurrent first-in-first-out queue. Elements are
// processed in the order they are added.
type Queue[InputT, OutputT any] struct {
	In  chan<- InputT
	Out <-chan OutputT

	inputs  chan InputT
	outputs chan OutputT
	doFunc  DoFunc[InputT, OutputT]
	running atomic.Bool
}

// DoFunc is the function that is called for each element in the queue.
type DoFunc[InputT, OutputT any] func(context.Context, InputT) OutputT

// NewQueue creates a new Queue.
func NewQueue[InputT, OutputT any](doFunc DoFunc[InputT, OutputT]) *Queue[InputT, OutputT] {
	q := &Queue[InputT, OutputT]{
		inputs:  make(chan InputT),
		outputs: make(chan OutputT, 1),
		doFunc:  doFunc,
	}
	q.In = q.inputs
	q.Out = q.outputs
	return q
}

// Run starts the queue. Run returns when the context is canceled.
func (q *Queue[InputT, OutputT]) Run(ctx context.Context) error {
	if !q.running.CompareAndSwap(false, true) {
		return errors.New("queue is already running")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case input := <-q.inputs:
			out := q.doFunc(ctx, input)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case q.outputs <- out:
			}
		}
	}
}

// Enqueue adds an element to the queue. If the queue is full, the function
// blocks until there is room for the element.
func (q *Queue[InputT, OutputT]) Enqueue(ctx context.Context, v ...InputT) error {
	for _, input := range v {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case q.inputs <- input:
		}
	}
	return nil
}

// Dequeue removes an element from the queue. If the queue is empty, the
// function blocks until there is an element.
func (q *Queue[InputT, OutputT]) Dequeue(ctx context.Context) (OutputT, error) {
	select {
	case <-ctx.Done():
		var z OutputT
		return z, ctx.Err()
	case output := <-q.outputs:
		return output, nil
	}
}

// TryDequeue removes an element from the queue. If the queue is empty, the
// function returns immediately.
func (q *Queue[InputT, OutputT]) TryDequeue() (OutputT, bool) {
	select {
	case output := <-q.outputs:
		return output, true
	default:
		var z OutputT
		return z, false
	}
}

// DequeueN removes n elements from the queue. If the queue is empty, the
// function blocks until there are n elements.
func (q *Queue[InputT, OutputT]) DequeueN(ctx context.Context, n int) ([]OutputT, error) {
	outputs := make([]OutputT, 0, n)
	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			return outputs, ctx.Err()
		case output := <-q.outputs:
			outputs = append(outputs, output)
		}
	}
	return outputs, nil
}
