/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package handlers

import (
	"encoding/json"
	"log"
	"mysql-graph-visualizer/internal/application/services/visualization"
	"mysql-graph-visualizer/internal/domain/valueobjects"
	"net/http"
)

type VisualizationHandler struct {
	service *visualization.VisualizationService
}

func NewVisualizationHandler(service *visualization.VisualizationService) *VisualizationHandler {
	return &VisualizationHandler{
		service: service,
	}
}

func (h *VisualizationHandler) GetGraphData(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()

	criteria := valueobjects.SearchCriteria{
		Labels: r.URL.Query()["labels"],
	}

	data, err := h.service.GetGraphData(ctx, criteria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func (h *VisualizationHandler) ExportGraph(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	format := r.URL.Query().Get("format")

	data, err := h.service.ExportGraph(ctx, format)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(data); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}

func (h *VisualizationHandler) GetConfig(w http.ResponseWriter, r *http.Request) {
	config := h.service.GetConfig()
	log.Printf("Config: %+v", config)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
		return
	}
}

// Handler implementation
