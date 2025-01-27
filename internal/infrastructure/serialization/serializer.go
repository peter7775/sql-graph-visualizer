/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

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
