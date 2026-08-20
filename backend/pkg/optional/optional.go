package optional

import "encoding/json"

// Generic wrapper to distinguish fields not present in JSON and explicitly set to null
// (e.g. Valid=true, Value=nil) for APIs that need to distinguish "no change" from "set to null".
type Optional[T any] struct {
	Value *T
	Valid bool
}

// Ptr returns a double-pointer: nil if the field was not provided,
// or a pointer to the value pointer (which may itself be nil for null).
func (o Optional[T]) Ptr() **T {
	if !o.Valid {
		return nil
	}
	return &o.Value
}

func (o *Optional[T]) UnmarshalJSON(data []byte) error {
	o.Valid = true
	if string(data) == "null" {
		o.Value = nil
		return nil
	}
	var v T
	if err := json.Unmarshal(data, &v); err != nil {
		return err
	}
	o.Value = &v
	return nil
}
