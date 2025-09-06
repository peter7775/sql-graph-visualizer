/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package handlers

import (
	"encoding/json"
	"mysql-graph-visualizer/internal/application/services/graph"
	"net/http"
)

type VisualizationHandler struct {
	neo4jURI      string
	neo4jUsername string
	neo4jPassword string
	graphService  graph.GraphService
}

func NewVisualizationHandler(uri, username, password string, graphService graph.GraphService) *VisualizationHandler {
	return &VisualizationHandler{
		neo4jURI:      uri,
		neo4jUsername: username,
		neo4jPassword: password,
		graphService:  graphService,
	}
}

func (h *VisualizationHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]any{
		"neo4j": map[string]string{
			"uri":      h.neo4jURI,
			"username": h.neo4jUsername,
			"password": h.neo4jPassword,
		},
	}
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func (h *VisualizationHandler) Search(w http.ResponseWriter, r *http.Request) {
	term := r.URL.Query().Get("term")
	results, err := h.graphService.SearchNodes(term)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(results); err != nil {
		http.Error(w, "Error encoding response", http.StatusInternalServerError)
		return
	}
}

func (h *VisualizationHandler) Export(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Format string `json:"format"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	switch req.Format {
	case "image":
		// Implementation of image export
		data, err := h.graphService.ExportImage()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		if _, err := w.Write(data); err != nil {
			http.Error(w, "Error writing response", http.StatusInternalServerError)
			return
		}
	case "json":
		// Export to JSON
		data, err := h.graphService.ExportJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(data); err != nil {
			http.Error(w, "Error encoding response", http.StatusInternalServerError)
			return
		}
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}
