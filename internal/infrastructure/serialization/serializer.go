/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package serialization

import "encoding/base64"

func SerializeByteArrayToString(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DeserializeStringToByteArray(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}
