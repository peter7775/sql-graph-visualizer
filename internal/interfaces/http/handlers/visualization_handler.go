package handlers

import (
	"github.com/peter7775/alevisualizer/internal/application/services/visualization"
	"github.com/peter7775/alevisualizer/internal/domain/valueobjects"
	"encoding/json"
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

	json.NewEncoder(w).Encode(data)
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
	json.NewEncoder(w).Encode(data)
}

// Implementace handlerů
