package serialization

import "fmt"

// SerializeID converts an ID to a string format or an array if it's a long type.
func SerializeID(id interface{}) interface{} {
	switch v := id.(type) {
	case int64:
		return []int64{v} // Převod na pole
	default:
		return fmt.Sprintf("%v", id)
	}
}
