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
	graphService  services.GraphService
}

func NewVisualizationHandler(uri, username, password string, graphService services.GraphService) *VisualizationHandler {
	return &VisualizationHandler{
		neo4jURI:      uri,
		neo4jUsername: username,
		neo4jPassword: password,
		graphService:  graphService,
	}
}

func (h *VisualizationHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := map[string]interface{}{
		"neo4j": map[string]string{
			"uri":      h.neo4jURI,
			"username": h.neo4jUsername,
			"password": h.neo4jPassword,
		},
	}
	json.NewEncoder(w).Encode(config)
}

func (h *VisualizationHandler) Search(w http.ResponseWriter, r *http.Request) {
	term := r.URL.Query().Get("term")
	results, err := h.graphService.SearchNodes(term)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	json.NewEncoder(w).Encode(results)
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
		// Implementace exportu do obrázku
		data, err := h.graphService.ExportImage()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "image/png")
		w.Write(data)
	case "json":
		// Export do JSON
		data, err := h.graphService.ExportJSON()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(data)
	default:
		http.Error(w, "Unsupported format", http.StatusBadRequest)
	}
}
