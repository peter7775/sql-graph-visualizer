/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package ports

type MySQLPort interface {
	FetchData() ([]map[string]interface{}, error)
	Close() error
	ExecuteQuery(query string) ([]map[string]interface{}, error)
}
