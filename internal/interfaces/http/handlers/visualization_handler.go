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

// Package handlers provides HTTP request handlers.
package handlers

import (
	"encoding/json"
	"log"
	"net/http"
	"sql-graph-visualizer/internal/application/services/visualization"
	"sql-graph-visualizer/internal/domain/valueobjects"
)

// VisualizationHandler handles visualization HTTP requests.
type VisualizationHandler struct {
	service *visualization.VisualizationService
}

// NewVisualizationHandler creates a new visualization handler.
func NewVisualizationHandler(service *visualization.VisualizationService) *VisualizationHandler {
	return &VisualizationHandler{
		service: service,
	}
}

// GetGraphData handles requests for graph data.
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

// ExportGraph handles graph export requests.
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

// GetConfig handles configuration requests.
func (h *VisualizationHandler) GetConfig(w http.ResponseWriter, _ *http.Request) {
	config := h.service.GetConfig()
	log.Printf("Config: %+v", config)
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(config); err != nil {
		http.Error(w, "Error encoding JSON", http.StatusInternalServerError)
		log.Printf("Error encoding JSON: %v", err)
		return
	}
}
