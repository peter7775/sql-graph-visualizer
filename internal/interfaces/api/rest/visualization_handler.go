package rest

import (
	"github.com/peter7775/alevisualizer/internal/application/dto"
	"github.com/peter7775/alevisualizer/internal/application/services"
	"net/http"
)

type VisualizationHandler struct {
	visualizationService *services.VisualizationService
}

func NewVisualizationHandler(service *services.VisualizationService) *VisualizationHandler {
	return &VisualizationHandler{
		visualizationService: service,
	}
}

func (h *VisualizationHandler) HandleVisualization(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	searchDTO := dto.SearchDTO{} // parse from request

	result, err := h.visualizationService.VisualizeGraph(ctx, searchDTO)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Return response
}
