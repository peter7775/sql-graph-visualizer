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

package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"sql-graph-visualizer/internal/application/services/visualization"
	"sql-graph-visualizer/internal/domain/valueobjects"
)

type VisualizationHandler struct {
	visualizationService *visualization.VisualizationService
}

func NewVisualizationHandler(service *visualization.VisualizationService) *VisualizationHandler {
	return &VisualizationHandler{
		visualizationService: service,
	}
}

func (h *VisualizationHandler) HandleVisualization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	criteria := valueobjects.SearchCriteria{} // Empty criteria for now
	result, err := h.visualizationService.GetGraphData(ctx, criteria)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Error encoding JSON: %v", err)
	}
}
