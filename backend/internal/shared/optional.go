package shared

import "encoding/json"

// Optional is a generic wrapper that distinguishes between a field not being
// present in JSON (zero value, Valid=false) and being explicitly set to null
// (Valid=true, Value=nil). Use Ptr() to convert to a double-pointer (**T)
// for APIs that need to distinguish "no change" from "set to null".
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

// UnmarshalJSON implements json.Unmarshaler.
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
