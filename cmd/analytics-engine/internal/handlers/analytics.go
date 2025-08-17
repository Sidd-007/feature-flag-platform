package handlers

import (
	"net/http"

	"github.com/rs/zerolog/log"

	"github.com/feature-flag-platform/cmd/analytics-engine/internal/middleware"
	"github.com/feature-flag-platform/cmd/analytics-engine/internal/services"
)

func (h *Handlers) RunTTest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Parse request body
	var req services.TTestRequest
	if err := parseJSONRequest(r, &req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if len(req.TreatmentData) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "treatment_data is required")
		return
	}
	if len(req.ControlData) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "control_data is required")
		return
	}

	log.Info().
		Str("user_id", userID).
		Int("treatment_size", len(req.TreatmentData)).
		Int("control_size", len(req.ControlData)).
		Float64("alpha", req.Alpha).
		Str("alternative", req.Alternative).
		Bool("equal_var", req.EqualVar).
		Msg("Running t-test")

	// Call service
	result, err := h.statsService.RunTTest(&req)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Failed to run t-test")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to run t-test")
		return
	}

	log.Info().
		Str("user_id", userID).
		Float64("t_statistic", result.Statistic).
		Float64("p_value", result.PValue).
		Bool("significant", result.IsSignificant).
		Float64("effect_size", result.EffectSize).
		Msg("T-test completed")

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *Handlers) RunChiSquareTest(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Parse request body
	var req services.ChiSquareRequest
	if err := parseJSONRequest(r, &req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if len(req.ObservedFrequencies) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "observed_frequencies is required")
		return
	}

	log.Info().
		Str("user_id", userID).
		Int("rows", len(req.ObservedFrequencies)).
		Int("cols", len(req.ObservedFrequencies[0])).
		Float64("alpha", req.Alpha).
		Msg("Running chi-square test")

	// Call service
	result, err := h.statsService.RunChiSquareTest(&req)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Failed to run chi-square test")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to run chi-square test")
		return
	}

	log.Info().
		Str("user_id", userID).
		Float64("chi_square", result.Statistic).
		Float64("p_value", result.PValue).
		Bool("significant", result.IsSignificant).
		Float64("cramer_v", result.CramerV).
		Msg("Chi-square test completed")

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *Handlers) RunSequentialAnalysis(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())

	// Parse request body
	var req services.SequentialAnalysisRequest
	if err := parseJSONRequest(r, &req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if len(req.TreatmentData) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "treatment_data is required")
		return
	}
	if len(req.ControlData) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "control_data is required")
		return
	}

	log.Info().
		Str("user_id", userID).
		Int("treatment_size", len(req.TreatmentData)).
		Int("control_size", len(req.ControlData)).
		Float64("alpha", req.Alpha).
		Float64("power", req.Power).
		Str("spending_function", req.SpendingFunction).
		Int("max_analyses", req.MaxAnalyses).
		Int("current_analysis", req.CurrentAnalysis).
		Msg("Running sequential analysis")

	// Call service
	result, err := h.statsService.RunSequentialAnalysis(&req)
	if err != nil {
		log.Error().Err(err).
			Str("user_id", userID).
			Msg("Failed to run sequential analysis")
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to run sequential analysis")
		return
	}

	log.Info().
		Str("user_id", userID).
		Float64("current_boundary", result.CurrentBoundary).
		Float64("test_statistic", result.TestStatistic).
		Bool("significant", result.IsSignificant).
		Bool("should_stop", result.ShouldStop).
		Msg("Sequential analysis completed")

	writeJSONResponse(w, http.StatusOK, result)
}

func (h *Handlers) AnalyzeFunnel(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse request body
	var requestBody struct {
		Steps []struct {
			Name       string            `json:"name"`
			MetricName string            `json:"metric_name"`
			Filters    map[string]string `json:"filters"`
		} `json:"steps"`
		TimeRange struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"time_range"`
		ExperimentID string            `json:"experiment_id"`
		Filters      map[string]string `json:"filters"`
	}

	if err := parseJSONRequest(r, &requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if len(requestBody.Steps) == 0 {
		writeErrorResponse(w, http.StatusBadRequest, "steps is required")
		return
	}

	log.Info().
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Int("steps", len(requestBody.Steps)).
		Str("experiment_id", requestBody.ExperimentID).
		Msg("Analyzing funnel")

	// TODO: Implement funnel analysis
	// For now, return placeholder response
	response := map[string]interface{}{
		"funnel_id":     "funnel_" + userID,
		"steps":         len(requestBody.Steps),
		"experiment_id": requestBody.ExperimentID,
		"status":        "analysis_in_progress",
		"message":       "Funnel analysis is being processed",
	}

	writeJSONResponse(w, http.StatusAccepted, response)
}

func (h *Handlers) AnalyzeCohort(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse request body
	var requestBody struct {
		CohortBy  string `json:"cohort_by"` // day, week, month
		TimeRange struct {
			Start string `json:"start"`
			End   string `json:"end"`
		} `json:"time_range"`
		MetricName   string            `json:"metric_name"`
		ExperimentID string            `json:"experiment_id"`
		Filters      map[string]string `json:"filters"`
	}

	if err := parseJSONRequest(r, &requestBody); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Validate required fields
	if requestBody.CohortBy == "" {
		writeErrorResponse(w, http.StatusBadRequest, "cohort_by is required")
		return
	}
	if requestBody.MetricName == "" {
		writeErrorResponse(w, http.StatusBadRequest, "metric_name is required")
		return
	}

	log.Info().
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Str("cohort_by", requestBody.CohortBy).
		Str("metric_name", requestBody.MetricName).
		Str("experiment_id", requestBody.ExperimentID).
		Msg("Analyzing cohort")

	// TODO: Implement cohort analysis
	// For now, return placeholder response
	response := map[string]interface{}{
		"cohort_id":     "cohort_" + userID,
		"cohort_by":     requestBody.CohortBy,
		"metric_name":   requestBody.MetricName,
		"experiment_id": requestBody.ExperimentID,
		"status":        "analysis_in_progress",
		"message":       "Cohort analysis is being processed",
	}

	writeJSONResponse(w, http.StatusAccepted, response)
}

func (h *Handlers) GetExposureMetrics(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse query parameters
	query := r.URL.Query()
	experimentID := query.Get("experiment_id")

	log.Info().
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Str("experiment_id", experimentID).
		Msg("Getting exposure metrics")

	// TODO: Implement exposure metrics calculation
	// For now, return placeholder response
	response := map[string]interface{}{
		"environment_id":  environmentID,
		"experiment_id":   experimentID,
		"total_exposures": 10000,
		"unique_users":    8500,
		"exposure_rate":   0.85,
		"daily_exposures": []map[string]interface{}{
			{"date": "2024-01-01", "exposures": 500, "unique_users": 425},
			{"date": "2024-01-02", "exposures": 520, "unique_users": 442},
		},
		"by_variation": map[string]interface{}{
			"control": map[string]interface{}{
				"exposures":    5000,
				"unique_users": 4250,
				"allocation":   0.5,
			},
			"treatment": map[string]interface{}{
				"exposures":    5000,
				"unique_users": 4250,
				"allocation":   0.5,
			},
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (h *Handlers) GetConversionMetrics(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse query parameters
	query := r.URL.Query()
	experimentID := query.Get("experiment_id")
	metricName := query.Get("metric_name")

	log.Info().
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Str("experiment_id", experimentID).
		Str("metric_name", metricName).
		Msg("Getting conversion metrics")

	// TODO: Implement conversion metrics calculation
	// For now, return placeholder response
	response := map[string]interface{}{
		"environment_id":          environmentID,
		"experiment_id":           experimentID,
		"metric_name":             metricName,
		"overall_conversion_rate": 0.15,
		"total_conversions":       1275,
		"daily_conversions": []map[string]interface{}{
			{"date": "2024-01-01", "conversions": 75, "rate": 0.15},
			{"date": "2024-01-02", "conversions": 78, "rate": 0.15},
		},
		"by_variation": map[string]interface{}{
			"control": map[string]interface{}{
				"conversions":     625,
				"conversion_rate": 0.125,
				"lift":            0.0,
			},
			"treatment": map[string]interface{}{
				"conversions":     650,
				"conversion_rate": 0.130,
				"lift":            0.04,
			},
		},
		"statistical_significance": map[string]interface{}{
			"p_value":        0.045,
			"is_significant": true,
			"confidence":     0.95,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}

func (h *Handlers) GetRetentionMetrics(w http.ResponseWriter, r *http.Request) {
	userID := middleware.GetUserID(r.Context())
	environmentID := middleware.GetEnvironmentID(r.Context())

	// Parse query parameters
	query := r.URL.Query()
	experimentID := query.Get("experiment_id")
	retentionBy := query.Get("retention_by")
	if retentionBy == "" {
		retentionBy = "day"
	}

	log.Info().
		Str("user_id", userID).
		Str("environment_id", environmentID).
		Str("experiment_id", experimentID).
		Str("retention_by", retentionBy).
		Msg("Getting retention metrics")

	// TODO: Implement retention metrics calculation
	// For now, return placeholder response
	response := map[string]interface{}{
		"environment_id": environmentID,
		"experiment_id":  experimentID,
		"retention_by":   retentionBy,
		"cohort_analysis": []map[string]interface{}{
			{
				"cohort_date": "2024-01-01",
				"cohort_size": 1000,
				"retention_periods": []map[string]interface{}{
					{"period": 1, "retained_users": 850, "retention_rate": 0.85},
					{"period": 7, "retained_users": 720, "retention_rate": 0.72},
					{"period": 30, "retained_users": 650, "retention_rate": 0.65},
				},
			},
		},
		"by_variation": map[string]interface{}{
			"control": map[string]interface{}{
				"day_1_retention":  0.84,
				"day_7_retention":  0.70,
				"day_30_retention": 0.62,
			},
			"treatment": map[string]interface{}{
				"day_1_retention":  0.86,
				"day_7_retention":  0.74,
				"day_30_retention": 0.68,
			},
		},
		"lift_analysis": map[string]interface{}{
			"day_1_lift":  0.024,
			"day_7_lift":  0.057,
			"day_30_lift": 0.097,
		},
	}

	writeJSONResponse(w, http.StatusOK, response)
}
