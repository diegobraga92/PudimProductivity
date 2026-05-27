package shared

import "encoding/json"

// Optional represents a value that may be absent, explicitly null, or set.
// This replaces the **string double-pointer pattern for partial updates.
//
//   - Absent:  set=false, val=nil  → field was absent; do not change the stored value.
//   - Null:    set=true, val=nil   → field was explicitly null; clear the stored value.
//   - Present: set=true, val=&s    → field was a string; set the stored value to s.
type Optional[T any] struct {
	set bool
	val *T
}

func NewOptional[T any](v T) Optional[T] {
	return Optional[T]{set: true, val: &v}
}

func NewNullOptional[T any]() Optional[T] {
	return Optional[T]{set: true, val: nil}
}

func NewAbsentOptional[T any]() Optional[T] {
	return Optional[T]{set: false, val: nil}
}

func (o Optional[T]) IsSet() bool {
	return o.set
}

func (o Optional[T]) IsNull() bool {
	return o.set && o.val == nil
}

func (o Optional[T]) Get() *T {
	if !o.set || o.val == nil {
		return nil
	}
	return o.val
}

// Ptr returns a **T suitable for use with the double-pointer pattern.
// nil outer pointer if absent, otherwise a pointer to val (which may be nil for null).
func (o Optional[T]) Ptr() **T {
	if !o.set {
		return nil
	}
	return &o.val
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.set = true
	if string(data) == "null" {
		o.val = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.val = &v
	return nil
}
