package concqueue

// SafeList provides non-panic access to a list.
type SafeList[T any] []T

// First returns the first element of the list.
func (s *SafeList[T]) First() *T {
	if len(*s) == 0 {
		return nil
	}
	return &(*s)[0]
}

// Last returns the last element of the list.
func (s *SafeList[T]) Last() *T {
	if len(*s) == 0 {
		return nil
	}
	return &(*s)[len(*s)-1]
}

// At returns the element at the given index. If the index is out of bounds,
// nil is returned.
func (s *SafeList[T]) At(i int) *T {
	if i < 0 || i >= len(*s) {
		return nil
	}
	return &(*s)[i]
}
