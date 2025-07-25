package service

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"event/internal/dao/repository"
)

// HTTPHandler provides HTTP endpoints for testing the business logic
type HTTPHandler struct {
	eventHandler *EventHandler
}

// NewHTTPHandler creates a new HTTP handler
func NewHTTPHandler(eventHandler *EventHandler) *HTTPHandler {
	return &HTTPHandler{
		eventHandler: eventHandler,
	}
}

// SetupRoutes sets up HTTP routes for testing
func (h *HTTPHandler) SetupRoutes(router *mux.Router) {
	// Console API routes
	console := router.PathPrefix("/console").Subrouter()
	console.Use(h.authMiddleware)
	
	console.HandleFunc("/events", h.CreateEvent).Methods("POST")
	console.HandleFunc("/events", h.GetEventList).Methods("GET")
	console.HandleFunc("/events/{id}", h.GetEvent).Methods("GET")
	console.HandleFunc("/events/{id}", h.UpdateEvent).Methods("PUT")
	console.HandleFunc("/events/{id}", h.PatchEvent).Methods("PATCH")
	console.HandleFunc("/events/{id}", h.DeleteEvent).Methods("DELETE")
	console.HandleFunc("/events/{id}/status", h.UpdateEventStatus).Methods("PUT")

	// Public API routes (no auth required)
	router.HandleFunc("/events", h.SearchPublicEvents).Methods("GET")
	router.HandleFunc("/events/{id}", h.GetPublicEvent).Methods("GET")
}

// Middleware to check required headers
func (h *HTTPHandler) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Check required headers
		userID := r.Header.Get("X-User-Id")
		brandID := r.Header.Get("X-Brand-Id")
		
		if userID == "" || brandID == "" {
			http.Error(w, "Missing required headers: X-User-Id, X-Brand-Id", http.StatusUnauthorized)
			return
		}
		
		next.ServeHTTP(w, r)
	})
}

// CreateEvent handles POST /console/events
func (h *HTTPHandler) CreateEvent(w http.ResponseWriter, r *http.Request) {
	var req CreateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	event, err := h.eventHandler.CreateEvent(r.Context(), &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   event,
	})
}

// GetEventList handles GET /console/events
func (h *HTTPHandler) GetEventList(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	filter := &repository.EventFilter{
		Limit: 20, // Default
	}

	if status := r.URL.Query().Get("status"); status != "" {
		filter.Status = &status
	}
	if visibility := r.URL.Query().Get("visibility"); visibility != "" {
		filter.Visibility = &visibility
	}
	if titleSearch := r.URL.Query().Get("title_search"); titleSearch != "" {
		filter.TitleSearch = &titleSearch
	}
	if sortBy := r.URL.Query().Get("sort_by"); sortBy != "" {
		filter.SortBy = &sortBy
	}
	if sortOrder := r.URL.Query().Get("sort_order"); sortOrder != "" {
		filter.SortOrder = &sortOrder
	}
	if pageToken := r.URL.Query().Get("page_token"); pageToken != "" {
		filter.PageToken = &pageToken
	}
	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		if size, err := strconv.Atoi(pageSize); err == nil && size > 0 && size <= 100 {
			filter.Limit = size
		}
	}
	if page := r.URL.Query().Get("page"); page != "" {
		if pageNum, err := strconv.Atoi(page); err == nil && pageNum > 0 {
			filter.Offset = (pageNum - 1) * filter.Limit
			filter.PageToken = nil // Don't use cursor if page is specified
		}
	}

	result, err := h.eventHandler.GetEventList(r.Context(), filter)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   result,
	})
}

// GetEvent handles GET /console/events/{id}
func (h *HTTPHandler) GetEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	event, err := h.eventHandler.GetEvent(r.Context(), eventID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   event,
	})
}

// UpdateEvent handles PUT /console/events/{id}
func (h *HTTPHandler) UpdateEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	var req UpdateEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	req.ID = eventID

	event, err := h.eventHandler.UpdateEvent(r.Context(), &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   event,
	})
}

// PatchEvent handles PATCH /console/events/{id}
func (h *HTTPHandler) PatchEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	var req PatchEventRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}
	req.ID = eventID

	event, err := h.eventHandler.PatchEvent(r.Context(), &req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   event,
	})
}

// DeleteEvent handles DELETE /console/events/{id}
func (h *HTTPHandler) DeleteEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	err := h.eventHandler.DeleteEvent(r.Context(), eventID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   nil,
	})
}

// UpdateEventStatus handles PUT /console/events/{id}/status
func (h *HTTPHandler) UpdateEventStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	event, err := h.eventHandler.UpdateEventStatus(r.Context(), eventID, req.Status)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   event,
	})
}

// SearchPublicEvents handles GET /events
func (h *HTTPHandler) SearchPublicEvents(w http.ResponseWriter, r *http.Request) {
	req := &SearchEventsRequest{}

	// Parse query parameters
	if brandID := r.URL.Query().Get("brand_id"); brandID != "" {
		req.BrandID = &brandID
	}
	if titleSearch := r.URL.Query().Get("title_search"); titleSearch != "" {
		req.TitleSearch = &titleSearch
	}
	if pageToken := r.URL.Query().Get("page_token"); pageToken != "" {
		req.PageToken = &pageToken
	}
	if page := r.URL.Query().Get("page"); page != "" {
		if pageNum, err := strconv.ParseInt(page, 10, 32); err == nil {
			p := int32(pageNum)
			req.Page = &p
		}
	}
	if pageSize := r.URL.Query().Get("page_size"); pageSize != "" {
		if size, err := strconv.ParseInt(pageSize, 10, 32); err == nil {
			s := int32(size)
			req.PageSize = &s
		}
	}

	result, err := h.eventHandler.SearchPublicEvents(r.Context(), req)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   result,
	})
}

// GetPublicEvent handles GET /events/{id}
func (h *HTTPHandler) GetPublicEvent(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	eventID := vars["id"]

	event, err := h.eventHandler.GetPublicEvent(r.Context(), eventID)
	if err != nil {
		h.handleError(w, err)
		return
	}

	h.writeResponse(w, map[string]interface{}{
		"status": "success",
		"code":   1000,
		"data":   event,
	})
}

// Helper methods

func (h *HTTPHandler) handleError(w http.ResponseWriter, err error) {
	grpcErr := h.eventHandler.HandleError(err)
	
	// Convert gRPC error to HTTP status
	statusCode := http.StatusInternalServerError
	switch {
	case grpcErr.Error() == "NotFound":
		statusCode = http.StatusNotFound
	case grpcErr.Error() == "InvalidArgument":
		statusCode = http.StatusBadRequest
	case grpcErr.Error() == "FailedPrecondition":
		statusCode = http.StatusConflict
	case grpcErr.Error() == "Unauthenticated":
		statusCode = http.StatusUnauthorized
	}

	h.writeErrorResponse(w, statusCode, err.Error())
}

func (h *HTTPHandler) writeResponse(w http.ResponseWriter, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(data)
}

func (h *HTTPHandler) writeErrorResponse(w http.ResponseWriter, statusCode int, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "error",
		"code":    statusCode,
		"message": message,
	})
}