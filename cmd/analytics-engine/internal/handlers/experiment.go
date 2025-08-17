package handlers

import (
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/rs/zerolog/log"

	"github.com/Sidd-007/feature-flag-platform/cmd/analytics-engine/internal/middleware"
	"github.com/Sidd-007/feature-flag-platform/cmd/analytics-engine/internal/repository"
	"github.com/Sidd-007/feature-flag-platform/cmd/analytics-engine/internal/services"
)

func (h *Handlers) GetExperimentResults(w http.ResponseWriter, r *http.Request) {
	experimentID := chi.URLParam(r, "experimentId")
	if experimentID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "experiment_id is required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse query parameters
	query := r.URL.Query()

	// Time range
	startTime, err := parseTimeParam(query.Get("start_time"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid start_time parameter")
		return
	}

	endTime, err := parseTimeParam(query.Get("end_time"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid end_time parameter")
		return
	}

	// Metric names
	metricNames := query["metric_name"]
	if len(metricNames) == 0 {
		metricNames = []string{"conversion", "revenue"} // Default metrics
	}

	// Other parameters
	includeRaw := query.Get("include_raw") == "true"
	includeCI := query.Get("include_ci") == "true"
	confidence, _ := strconv.ParseFloat(query.Get("confidence"), 64)
	if confidence == 0 {
		confidence = 0.95
	}

	// Build request
	req := &services.ExperimentResultsRequest{
		ExperimentID:  experimentID,
		EnvironmentID: environmentID,
		TimeRange: repository.TimeRange{
			Start: startTime,
			End:   endTime,
		},
		MetricNames: metricNames,
		IncludeRaw:  includeRaw,
		IncludeCI:   includeCI,
		Confidence:  confidence,
		Filters:     make(map[string]string),
	}

	// Add filters from query parameters
	for key, values := range query {
		if len(values) > 0 && key != "start_time" && key != "end_time" &&
			key != "metric_name" && key != "include_raw" && key != "include_ci" && key != "confidence" {
			req.Filters[key] = values[0]
		}
	}

	log.Info().
		Str("experiment_id", experimentID).
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Strs("metric_names", metricNames).
		Time("start_time", startTime).
		Time("end_time", endTime).
		Msg("Getting experiment results")

	// Call service
	results, err := h.experimentService.GetExperimentResults(r.Context(), req)
	if err != nil {
		log.Error().Err(err).
			Str("experiment_id", experimentID).
			Msg("Failed to get experiment results")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get experiment results")
		return
	}

	writeJSONResponse(w, http.StatusOK, results)
}

func (h *Handlers) GetExperimentSummary(w http.ResponseWriter, r *http.Request) {
	experimentID := chi.URLParam(r, "experimentId")
	if experimentID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "experiment_id is required")
		return
	}

	userID := middleware.GetUserID(r.Context())

	// Parse query parameters
	query := r.URL.Query()

	startTime, err := parseTimeParam(query.Get("start_time"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid start_time parameter")
		return
	}

	endTime, err := parseTimeParam(query.Get("end_time"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid end_time parameter")
		return
	}

	timeRange := repository.TimeRange{
		Start: startTime,
		End:   endTime,
	}

	log.Info().
		Str("experiment_id", experimentID).
		Str("user_id", userID).
		Time("start_time", startTime).
		Time("end_time", endTime).
		Msg("Getting experiment summary")

	// Call service
	summary, err := h.experimentService.GetExperimentSummary(r.Context(), experimentID, timeRange)
	if err != nil {
		log.Error().Err(err).
			Str("experiment_id", experimentID).
			Msg("Failed to get experiment summary")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get experiment summary")
		return
	}

	writeJSONResponse(w, http.StatusOK, summary)
}

func (h *Handlers) GetExperimentTimeline(w http.ResponseWriter, r *http.Request) {
	experimentID := chi.URLParam(r, "experimentId")
	if experimentID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "experiment_id is required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse query parameters
	query := r.URL.Query()

	startTime, err := parseTimeParam(query.Get("start_time"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid start_time parameter")
		return
	}

	endTime, err := parseTimeParam(query.Get("end_time"))
	if err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "invalid end_time parameter")
		return
	}

	granularity := query.Get("granularity")
	if granularity == "" {
		granularity = "day"
	}

	metricNames := query["metric_name"]
	if len(metricNames) == 0 {
		metricNames = []string{"conversion"}
	}

	req := &services.ExperimentTimelineRequest{
		ExperimentID:  experimentID,
		EnvironmentID: environmentID,
		TimeRange: repository.TimeRange{
			Start: startTime,
			End:   endTime,
		},
		Granularity: granularity,
		MetricNames: metricNames,
	}

	log.Info().
		Str("experiment_id", experimentID).
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Str("granularity", granularity).
		Strs("metric_names", metricNames).
		Msg("Getting experiment timeline")

	// Call service
	timeline, err := h.experimentService.GetExperimentTimeline(r.Context(), req)
	if err != nil {
		log.Error().Err(err).
			Str("experiment_id", experimentID).
			Msg("Failed to get experiment timeline")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get experiment timeline")
		return
	}

	writeJSONResponse(w, http.StatusOK, timeline)
}

func (h *Handlers) AnalyzeExperiment(w http.ResponseWriter, r *http.Request) {
	experimentID := chi.URLParam(r, "experimentId")
	if experimentID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "experiment_id is required")
		return
	}

	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse request body
	var requestBody struct {
		TimeRange        repository.TimeRange `json:"time_range"`
		PrimaryMetric    string               `json:"primary_metric"`
		SecondaryMetrics []string             `json:"secondary_metrics"`
		Alpha            float64              `json:"alpha"`
		Power            float64              `json:"power"`
		MinEffect        float64              `json:"min_effect"`
		Sequential       bool                 `json:"sequential"`
		UseCUPED         bool                 `json:"use_cuped"`
		CovariateMetric  string               `json:"covariate_metric"`
	}

	if err := parseJSONRequest(r, &requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if requestBody.PrimaryMetric == "" {
		writeErrorResponse(w, http.StatusBadRequest, "primary_metric is required")
		return
	}

	req := &services.ExperimentAnalysisRequest{
		ExperimentID:     experimentID,
		EnvironmentID:    environmentID,
		TimeRange:        requestBody.TimeRange,
		PrimaryMetric:    requestBody.PrimaryMetric,
		SecondaryMetrics: requestBody.SecondaryMetrics,
		Alpha:            requestBody.Alpha,
		Power:            requestBody.Power,
		MinEffect:        requestBody.MinEffect,
		Sequential:       requestBody.Sequential,
		UseCUPED:         requestBody.UseCUPED,
		CovariateMetric:  requestBody.CovariateMetric,
	}

	log.Info().
		Str("experiment_id", experimentID).
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Str("primary_metric", requestBody.PrimaryMetric).
		Int("secondary_metrics", len(requestBody.SecondaryMetrics)).
		Bool("sequential", requestBody.Sequential).
		Bool("use_cuped", requestBody.UseCUPED).
		Msg("Running experiment analysis")

	// Call service
	analysis, err := h.experimentService.AnalyzeExperiment(r.Context(), req)
	if err != nil {
		log.Error().Err(err).
			Str("experiment_id", experimentID).
			Msg("Failed to analyze experiment")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to analyze experiment")
		return
	}

	writeJSONResponse(w, http.StatusOK, analysis)
}

// Helper functions
func parseTimeParam(timeStr string) (time.Time, error) {
	if timeStr == "" {
		// Default to last 30 days
		return time.Now().AddDate(0, 0, -30), nil
	}

	// Try different time formats
	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, timeStr); err == nil {
			return t, nil
		}
	}

	// Try parsing as Unix timestamp
	if timestamp, err := strconv.ParseInt(timeStr, 10, 64); err == nil {
		return time.Unix(timestamp, 0), nil
	}

	return time.Time{}, nil
}
