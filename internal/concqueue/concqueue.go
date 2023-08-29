package concqueue

import (
	"context"
	"errors"
	"sync/atomic"
)

// Result is the result of a single queue operation.
type Result[T any] struct {
	v   T
	err error
}

// Ok returns a successful result.
func Ok[T any](v T) Result[T] {
	return Result[T]{v: v}
}

// Error returns an error result.
func Error[T any](err error) Result[T] {
	return Result[T]{err: err}
}

// NoResult returns a result that indicates that there is no result. This is
// useful for queues that do not return a result.
func NoResult[T any]() Result[T] {
	return Result[T]{err: errNoResult}
}

var errNoResult = errors.New("no result")

// Value returns the value of the result. It panics if the result is an error.
func (r *Result[T]) Value() T {
	if r.err != nil {
		panic(r.err)
	}
	return r.v
}

// Error returns the error of the result. It returns nil if the result is not an
// error.
func (r *Result[T]) Error() error {
	return r.err
}

// Queue implements a concurrent first-in-first-out queue. Elements are
// processed in the order they are added.
type Queue[InputT, OutputT any] struct {
	In  chan<- InputT
	Out <-chan Result[OutputT]

	inputs  chan InputT
	outputs chan Result[OutputT]
	doFunc  DoFunc[InputT, OutputT]
	running atomic.Bool
}

// DoFunc is the function that is called for each element in the queue.
type DoFunc[InputT, OutputT any] func(context.Context, InputT) (OutputT, error)

// NewQueue creates a new Queue.
func NewQueue[InputT, OutputT any](doFunc DoFunc[InputT, OutputT]) *Queue[InputT, OutputT] {
	q := &Queue[InputT, OutputT]{
		inputs:  make(chan InputT),
		outputs: make(chan Result[OutputT], 1),
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
			out, err := q.doFunc(ctx, input)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case q.outputs <- Result[OutputT]{v: out, err: err}:
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

// EnqueueList adds a list of elements to the queue. If the queue is full, the
// function blocks until there is room for the elements.
func (q *Queue[InputT, OutputT]) EnqueueList(ctx context.Context, vs []InputT) error {
	return q.Enqueue(ctx, vs...)
}

// Dequeue removes an element from the queue. If the queue is empty, the
// function blocks until there is an element.
func (q *Queue[InputT, OutputT]) Dequeue(ctx context.Context) (Result[OutputT], error) {
	select {
	case <-ctx.Done():
		return Result[OutputT]{err: errNoResult}, ctx.Err()
	case output := <-q.outputs:
		return output, nil
	}
}

// TryDequeue removes an element from the queue. If the queue is empty, the
// function returns immediately with a zero value.
func (q *Queue[InputT, OutputT]) TryDequeue() (Result[OutputT], bool) {
	select {
	case output := <-q.outputs:
		return output, true
	default:
		return NoResult[OutputT](), false
	}
}

// DequeueN removes n elements from the queue. If the queue is empty, the
// function blocks until there are n elements.
func (q *Queue[InputT, OutputT]) DequeueN(ctx context.Context, n int) ([]Result[OutputT], error) {
	outputs := make([]Result[OutputT], 0, n)
	for i := 0; i < n; i++ {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case output := <-q.Out:
			outputs = append(outputs, output)
		}
	}
	return outputs, nil
}
