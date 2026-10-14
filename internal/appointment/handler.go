package appointment

import (
	"encoding/json"
	"errors"
	"net/http"

	"github.com/example/healthtech-image-ops-service/internal/infrai"
)

type optimizeHandler struct {
	service *Service
}

func NewOptimizeHandler(service *Service) http.Handler {
	return &optimizeHandler{service: service}
}

func (h *optimizeHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}

	var req OptimizeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "invalid JSON body", http.StatusBadRequest)
		return
	}

	if err := validateRequest(req); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	result, err := h.service.OptimizeAppointmentImage(r.Context(), req)
	if err != nil {
		var apiErr *infrai.APIError
		if errors.As(err, &apiErr) && apiErr.StatusCode >= 400 && apiErr.StatusCode < 500 {
			http.Error(w, apiErr.Message, http.StatusBadRequest)
			return
		}
		http.Error(w, "request failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(result)
}

func validateRequest(req OptimizeRequest) error {
	if req.AppointmentID == "" {
		return errors.New("appointment_id is required")
	}
	if req.PatientID == "" {
		return errors.New("patient_id is required")
	}
	if req.ContentType == "" {
		return errors.New("content_type is required")
	}
	if req.Filename == "" {
		return errors.New("filename is required")
	}
	if req.ImageBase64 == "" {
		return errors.New("image_base64 is required")
	}
	return nil
}
