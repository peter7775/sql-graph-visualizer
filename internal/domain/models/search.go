/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package models

// SearchResult represents the result of a graph search operation.
type SearchResult struct {
	ID     string   `json:"id"`
	Name   string   `json:"name"`
	Labels []string `json:"labels"`
}
