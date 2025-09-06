/*
 * Copyright (c) 2025 Petr Miroslav Stepanek <petrstepanek99@gmail.com>
 *
 * This source code is licensed under the MIT license found in the
 * LICENSE file in the root directory of this source tree.
 */

package rest

import (
	"encoding/json"
	"log"
	"mysql-graph-visualizer/internal/application/services/visualization"
	"mysql-graph-visualizer/internal/domain/valueobjects"
	"net/http"
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
