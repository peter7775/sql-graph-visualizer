package serialization

import "encoding/base64"

// SerializeByteArrayToString converts a byte array to a base64 encoded string.
func SerializeByteArrayToString(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

// DeserializeStringToByteArray converts a base64 encoded string back to a byte array.
func DeserializeStringToByteArray(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}
