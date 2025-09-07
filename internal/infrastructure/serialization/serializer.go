/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under a Dual License:
 * - AGPL-3.0 for open source use (see LICENSE file)
 * - Commercial License for business use (contact: petrstepanek99@gmail.com)
 *
 * This software contains patent-pending innovations in database analysis
 * and graph visualization. Commercial use requires separate licensing.
 */

package serialization

import "encoding/base64"

func SerializeByteArrayToString(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func DeserializeStringToByteArray(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}
