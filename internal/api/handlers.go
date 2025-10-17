package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"github.com/harishb93/telemetry-pipeline/internal/collector"
)

// Handlers contains HTTP request handlers for the API
type Handlers struct {
	collector    *collector.Collector
	collectorURL string // URL to the collector service
}

// NewHandlers creates a new handlers instance
func NewHandlers(collector *collector.Collector) *Handlers {
	// Get collector URL from environment, default to localhost for local development
	collectorURL := os.Getenv("COLLECTOR_URL")
	if collectorURL == "" {
		collectorURL = "http://telemetry-collector:8080"
	}

	return &Handlers{
		collector:    collector,
		collectorURL: collectorURL,
	}
}

// CollectorStats represents the stats returned by the collector service
type CollectorStats struct {
	GPUEntryCounts   map[string]int `json:"gpu_entry_counts"`
	MaxEntriesPerGPU int            `json:"max_entries_per_gpu"`
	TotalEntries     int            `json:"total_entries"`
	TotalGPUs        int            `json:"total_gpus"`
}

// GPUResponse represents the response for GPU list endpoint
type GPUResponse struct {
	GPUs       []string           `json:"gpus"`
	Total      int                `json:"total"`
	Pagination PaginationMetadata `json:"pagination"`
}

// TelemetryResponse represents the response for telemetry endpoint
type TelemetryResponse struct {
	Data       []*collector.Telemetry `json:"data"`
	Total      int                    `json:"total"`
	Pagination PaginationMetadata     `json:"pagination"`
}

// PaginationMetadata represents pagination information
type PaginationMetadata struct {
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasNext bool `json:"has_next"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

// @title Telemetry API
// @version 1.0
// @description API for accessing GPU telemetry data
// @termsOfService http://swagger.io/terms/
// @contact.name API Support
// @contact.email support@example.com
// @license.name MIT
// @license.url https://opensource.org/licenses/MIT
// @host localhost:8081
// @BasePath /api/v1

// GetGPUs returns a list of all GPUs with available telemetry data
// @Summary Get all GPU IDs
// @Description Returns a list of all GPU IDs for which telemetry data is available
// @Tags GPUs
// @Accept json
// @Produce json
// @Param limit query int false "Number of items to return (default: 50, max: 1000)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Success 200 {object} GPUResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /gpus [get]
func (h *Handlers) GetGPUs(w http.ResponseWriter, r *http.Request) {
	// Parse pagination parameters
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid pagination parameters", err.Error())
		return
	}

	// Get GPU IDs from both memory and file storage
	gpuIDs, err := h.getAllGPUIDs()
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve GPU IDs", err.Error())
		return
	}

	// Apply pagination
	total := len(gpuIDs)
	end := offset + limit
	if end > total {
		end = total
	}

	var paginatedGPUs []string
	if offset < total {
		paginatedGPUs = gpuIDs[offset:end]
	} else {
		paginatedGPUs = []string{}
	}

	response := GPUResponse{
		GPUs:  paginatedGPUs,
		Total: total,
		Pagination: PaginationMetadata{
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// GetTelemetry returns telemetry data for a specific GPU
// @Summary Get telemetry data for a GPU
// @Description Returns telemetry entries for a specific GPU, optionally filtered by time range
// @Tags Telemetry
// @Accept json
// @Produce json
// @Param id path string true "GPU ID"
// @Param start_time query string false "Start time filter (RFC3339 format)"
// @Param end_time query string false "End time filter (RFC3339 format)"
// @Param limit query int false "Number of items to return (default: 100, max: 1000)"
// @Param offset query int false "Number of items to skip (default: 0)"
// @Success 200 {object} TelemetryResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Router /gpus/{id}/telemetry [get]
func (h *Handlers) GetTelemetry(w http.ResponseWriter, r *http.Request) {
	// Extract GPU ID from URL path
	vars := mux.Vars(r)
	gpuID := vars["id"]

	if gpuID == "" {
		h.writeErrorResponse(w, http.StatusBadRequest, "Missing GPU ID", "GPU ID is required")
		return
	}

	// Parse pagination parameters
	limit, offset, err := h.parsePagination(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid pagination parameters", err.Error())
		return
	}

	// Parse time range parameters
	startTime, endTime, err := h.parseTimeRange(r)
	if err != nil {
		h.writeErrorResponse(w, http.StatusBadRequest, "Invalid time range parameters", err.Error())
		return
	}

	// Get telemetry data
	telemetryData, err := h.getTelemetryData(gpuID, startTime, endTime, limit, offset)
	if err != nil {
		if err.Error() == "GPU not found" {
			h.writeErrorResponse(w, http.StatusNotFound, "GPU not found", "No telemetry data found for GPU ID: "+gpuID)
			return
		}
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to retrieve telemetry data", err.Error())
		return
	}

	// Get total count (without pagination)
	totalData, err := h.getTelemetryData(gpuID, startTime, endTime, 0, 0)
	if err != nil {
		h.writeErrorResponse(w, http.StatusInternalServerError, "Failed to get total count", err.Error())
		return
	}
	total := len(totalData)

	response := TelemetryResponse{
		Data:  telemetryData,
		Total: total,
		Pagination: PaginationMetadata{
			Limit:   limit,
			Offset:  offset,
			HasNext: offset+limit < total,
		},
	}

	h.writeJSONResponse(w, http.StatusOK, response)
}

// Health returns the health status of the API
// @Summary Health check
// @Description Returns the health status of the API service
// @Tags Health
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /health [get]
func (h *Handlers) Health(w http.ResponseWriter, r *http.Request) {
	health := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().UTC(),
		"version":   "1.0.0",
		"service":   "telemetry-api",
	}

	// Add collector health status by fetching from collector service
	if stats, err := h.getCollectorStats(); err == nil {
		health["collector"] = map[string]interface{}{
			"status": "healthy",
			"memory_stats": map[string]interface{}{
				"gpu_entry_counts":    stats.GPUEntryCounts,
				"max_entries_per_gpu": stats.MaxEntriesPerGPU,
				"total_entries":       stats.TotalEntries,
				"total_gpus":          stats.TotalGPUs,
			},
		}
	} else {
		health["collector"] = map[string]interface{}{
			"status": "unhealthy",
			"error":  err.Error(),
		}
	}

	h.writeJSONResponse(w, http.StatusOK, health)
}

// Helper methods

func (h *Handlers) parsePagination(r *http.Request) (limit, offset int, err error) {
	// Parse limit
	limitStr := r.URL.Query().Get("limit")
	if limitStr == "" {
		limit = 100 // Default limit
	} else {
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			return 0, 0, err
		}
		if limit <= 0 || limit > 1000 {
			limit = 100 // Default if invalid
		}
	}

	// Parse offset
	offsetStr := r.URL.Query().Get("offset")
	if offsetStr == "" {
		offset = 0 // Default offset
	} else {
		offset, err = strconv.Atoi(offsetStr)
		if err != nil {
			return 0, 0, err
		}
		if offset < 0 {
			offset = 0 // Default if invalid
		}
	}

	return limit, offset, nil
}

func (h *Handlers) parseTimeRange(r *http.Request) (*time.Time, *time.Time, error) {
	var startTime, endTime *time.Time

	startTimeStr := r.URL.Query().Get("start_time")
	if startTimeStr != "" {
		t, err := time.Parse(time.RFC3339, startTimeStr)
		if err != nil {
			return nil, nil, err
		}
		startTime = &t
	}

	endTimeStr := r.URL.Query().Get("end_time")
	if endTimeStr != "" {
		t, err := time.Parse(time.RFC3339, endTimeStr)
		if err != nil {
			return nil, nil, err
		}
		endTime = &t
	}

	return startTime, endTime, nil
}

// getCollectorStats fetches stats from the collector service via HTTP
func (h *Handlers) getCollectorStats() (*CollectorStats, error) {
	resp, err := http.Get(h.collectorURL + "/stats")
	if err != nil {
		return nil, fmt.Errorf("failed to call collector stats endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collector stats endpoint returned status %d", resp.StatusCode)
	}

	var stats CollectorStats
	if err := json.NewDecoder(resp.Body).Decode(&stats); err != nil {
		return nil, fmt.Errorf("failed to decode collector stats response: %w", err)
	}

	return &stats, nil
}

func (h *Handlers) getAllGPUIDs() ([]string, error) {
	// Get GPU IDs from collector service via HTTP
	stats, err := h.getCollectorStats()
	if err != nil {
		return nil, fmt.Errorf("failed to get collector stats: %w", err)
	}

	var gpuIDs []string
	for gpuID := range stats.GPUEntryCounts {
		gpuIDs = append(gpuIDs, gpuID)
	}

	return gpuIDs, nil
}

func (h *Handlers) getTelemetryData(gpuID string, startTime, endTime *time.Time, limit, offset int) ([]*collector.Telemetry, error) {
	// Get telemetry data from collector service via HTTP
	url := fmt.Sprintf("%s/api/v1/gpus/%s/telemetry", h.collectorURL, gpuID)
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to call collector telemetry endpoint: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("collector telemetry endpoint returned status %d", resp.StatusCode)
	}

	var response struct {
		Data  []*collector.Telemetry `json:"data"`
		Total int                    `json:"total"`
		GpuID string                 `json:"gpu_id"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode collector telemetry response: %w", err)
	}

	allData := response.Data
	if len(allData) == 0 {
		return nil, nil // No error, just no data
	}

	// Apply time range filtering
	var filteredData []*collector.Telemetry
	for _, telemetry := range allData {
		include := true

		if startTime != nil && telemetry.Timestamp.Before(*startTime) {
			include = false
		}

		if endTime != nil && telemetry.Timestamp.After(*endTime) {
			include = false
		}

		if include {
			filteredData = append(filteredData, telemetry)
		}
	}

	// Apply pagination
	total := len(filteredData)
	if limit == 0 {
		return filteredData, nil // Return all if no limit specified
	}

	end := offset + limit
	if end > total {
		end = total
	}

	if offset >= total {
		return []*collector.Telemetry{}, nil // Return empty slice if offset is beyond data
	}

	return filteredData[offset:end], nil
}

func (h *Handlers) writeJSONResponse(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if err := json.NewEncoder(w).Encode(data); err != nil {
		// If we can't encode the response, log the error and send a generic error
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

func (h *Handlers) writeErrorResponse(w http.ResponseWriter, statusCode int, message, details string) {
	errorResp := ErrorResponse{
		Error:   http.StatusText(statusCode),
		Message: message,
		Code:    statusCode,
	}

	if details != "" {
		errorResp.Message = message + ": " + details
	}

	h.writeJSONResponse(w, statusCode, errorResp)
}
