package services

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/Sidd-007/feature-flag-platform/cmd/analytics-engine/internal/repository"
)

type ExperimentService interface {
	GetExperimentResults(ctx context.Context, req *ExperimentResultsRequest) (*ExperimentResults, error)
	GetExperimentSummary(ctx context.Context, experimentID string, timeRange repository.TimeRange) (*repository.ExperimentSummary, error)
	GetExperimentTimeline(ctx context.Context, req *ExperimentTimelineRequest) (*ExperimentTimeline, error)
	AnalyzeExperiment(ctx context.Context, req *ExperimentAnalysisRequest) (*ExperimentAnalysis, error)
}

type experimentService struct {
	eventRepo    repository.EventRepository
	statsService StatisticsService
}

func NewExperimentService(eventRepo repository.EventRepository, statsService StatisticsService) ExperimentService {
	return &experimentService{
		eventRepo:    eventRepo,
		statsService: statsService,
	}
}

// Request types
type ExperimentResultsRequest struct {
	ExperimentID  string               `json:"experiment_id"`
	EnvironmentID string               `json:"environment_id"`
	TimeRange     repository.TimeRange `json:"time_range"`
	MetricNames   []string             `json:"metric_names"`
	IncludeRaw    bool                 `json:"include_raw"` // Include raw event data
	IncludeCI     bool                 `json:"include_ci"`  // Include confidence intervals
	Confidence    float64              `json:"confidence"`  // Confidence level (default: 0.95)
	Filters       map[string]string    `json:"filters"`
}

type ExperimentTimelineRequest struct {
	ExperimentID  string               `json:"experiment_id"`
	EnvironmentID string               `json:"environment_id"`
	TimeRange     repository.TimeRange `json:"time_range"`
	Granularity   string               `json:"granularity"` // "hour", "day", "week"
	MetricNames   []string             `json:"metric_names"`
}

type ExperimentAnalysisRequest struct {
	ExperimentID     string               `json:"experiment_id"`
	EnvironmentID    string               `json:"environment_id"`
	TimeRange        repository.TimeRange `json:"time_range"`
	PrimaryMetric    string               `json:"primary_metric"`
	SecondaryMetrics []string             `json:"secondary_metrics"`
	Alpha            float64              `json:"alpha"`            // Significance level
	Power            float64              `json:"power"`            // Desired power
	MinEffect        float64              `json:"min_effect"`       // Minimum detectable effect
	Sequential       bool                 `json:"sequential"`       // Use sequential analysis
	UseCUPED         bool                 `json:"use_cuped"`        // Apply CUPED variance reduction
	CovariateMetric  string               `json:"covariate_metric"` // Metric to use as covariate for CUPED
}

// Response types
type ExperimentResults struct {
	ExperimentID     string                            `json:"experiment_id"`
	Summary          *repository.ExperimentSummary     `json:"summary"`
	MetricResults    map[string]*MetricAnalysis        `json:"metric_results"`
	VariationResults map[string]*VariationResults      `json:"variation_results"`
	StatisticalTests map[string]*StatisticalTestResult `json:"statistical_tests"`
	Recommendations  *ExperimentRecommendations        `json:"recommendations"`
	TimeRange        repository.TimeRange              `json:"time_range"`
	LastUpdated      time.Time                         `json:"last_updated"`
}

type MetricAnalysis struct {
	MetricName          string                         `json:"metric_name"`
	OverallStats        *DescriptiveStats              `json:"overall_stats"`
	ByVariation         map[string]*DescriptiveStats   `json:"by_variation"`
	TTestResults        map[string]*TTestResult        `json:"t_test_results"` // variation_id -> result
	ConfidenceIntervals map[string]*ConfidenceInterval `json:"confidence_intervals"`
	EffectSizes         map[string]float64             `json:"effect_sizes"`
	PowerAnalysis       *PowerAnalysis                 `json:"power_analysis"`
}

type VariationResults struct {
	VariationID     string                       `json:"variation_id"`
	ExposureCount   int64                        `json:"exposure_count"`
	UniqueUsers     int64                        `json:"unique_users"`
	ConversionRate  float64                      `json:"conversion_rate"`
	AllocationRatio float64                      `json:"allocation_ratio"`
	MetricSummaries map[string]*DescriptiveStats `json:"metric_summaries"`
}

type StatisticalTestResult struct {
	TestType      string      `json:"test_type"` // "t_test", "chi_square", "sequential"
	PValue        float64     `json:"p_value"`
	IsSignificant bool        `json:"is_significant"`
	EffectSize    float64     `json:"effect_size"`
	Confidence    float64     `json:"confidence"`
	Details       interface{} `json:"details"` // Test-specific details
}

type ExperimentRecommendations struct {
	Decision        string          `json:"decision"`   // "launch", "iterate", "stop"
	Confidence      string          `json:"confidence"` // "high", "medium", "low"
	Reasoning       []string        `json:"reasoning"`
	RiskAssessment  string          `json:"risk_assessment"`
	NextSteps       []string        `json:"next_steps"`
	EstimatedImpact *ImpactEstimate `json:"estimated_impact"`
}

type ImpactEstimate struct {
	RelativeLift    float64    `json:"relative_lift"`
	AbsoluteImpact  float64    `json:"absolute_impact"`
	ConfidenceRange [2]float64 `json:"confidence_range"`
	AnnualizedValue float64    `json:"annualized_value,omitempty"`
}

type ExperimentTimeline struct {
	ExperimentID string                    `json:"experiment_id"`
	Granularity  string                    `json:"granularity"`
	TimePoints   []TimePointData           `json:"time_points"`
	Trends       map[string]*TrendAnalysis `json:"trends"`
}

type TimePointData struct {
	Timestamp      time.Time                     `json:"timestamp"`
	ExposureCount  int64                         `json:"exposure_count"`
	UniqueUsers    int64                         `json:"unique_users"`
	MetricValues   map[string]map[string]float64 `json:"metric_values"` // metric_name -> variation_id -> value
	RunningResults map[string]*RunningTestResult `json:"running_results"`
}

type TrendAnalysis struct {
	MetricName     string           `json:"metric_name"`
	TrendDirection string           `json:"trend_direction"` // "increasing", "decreasing", "stable"
	TrendStrength  float64          `json:"trend_strength"`  // -1 to 1
	ChangePoints   []time.Time      `json:"change_points"`
	Seasonality    *SeasonalityInfo `json:"seasonality,omitempty"`
}

type SeasonalityInfo struct {
	Period    string   `json:"period"` // "daily", "weekly"
	Strength  float64  `json:"strength"`
	PeakTimes []string `json:"peak_times"`
}

type RunningTestResult struct {
	PValue        float64 `json:"p_value"`
	IsSignificant bool    `json:"is_significant"`
	EffectSize    float64 `json:"effect_size"`
	SampleSize    int     `json:"sample_size"`
}

type ExperimentAnalysis struct {
	ExperimentID      string                     `json:"experiment_id"`
	AnalysisType      string                     `json:"analysis_type"` // "frequentist", "sequential"
	PrimaryResults    *MetricAnalysis            `json:"primary_results"`
	SecondaryResults  map[string]*MetricAnalysis `json:"secondary_results"`
	OverallConclusion *ExperimentConclusion      `json:"overall_conclusion"`
	QualityChecks     *QualityAssessment         `json:"quality_checks"`
	AnalysisTimestamp time.Time                  `json:"analysis_timestamp"`
}

type ExperimentConclusion struct {
	Winner         string          `json:"winner"` // variation_id or "inconclusive"
	Confidence     float64         `json:"confidence"`
	ExpectedLift   float64         `json:"expected_lift"`
	RiskAssessment string          `json:"risk_assessment"`
	Recommendation string          `json:"recommendation"`
	BusinessImpact *BusinessImpact `json:"business_impact"`
}

type BusinessImpact struct {
	EstimatedLift       float64    `json:"estimated_lift"`
	LiftConfidenceRange [2]float64 `json:"lift_confidence_range"`
	RevenueImpact       float64    `json:"revenue_impact,omitempty"`
	UserImpact          int64      `json:"user_impact,omitempty"`
}

type QualityAssessment struct {
	SampleSizeAdequacy   string   `json:"sample_size_adequacy"` // "adequate", "marginal", "inadequate"
	BalanceCheck         string   `json:"balance_check"`        // "balanced", "slight_imbalance", "imbalanced"
	DataQualityIssues    []string `json:"data_quality_issues"`
	BiasAssessment       string   `json:"bias_assessment"`
	ExternalValidityRisk string   `json:"external_validity_risk"`
}

// Implementation
func (s *experimentService) GetExperimentResults(ctx context.Context, req *ExperimentResultsRequest) (*ExperimentResults, error) {
	log.Info().
		Str("experiment_id", req.ExperimentID).
		Str("environment_id", req.EnvironmentID).
		Strs("metric_names", req.MetricNames).
		Msg("Getting experiment results")

	// Set defaults
	if req.Confidence == 0 {
		req.Confidence = 0.95
	}

	// Get experiment summary
	summary, err := s.eventRepo.GetExperimentSummary(ctx, req.ExperimentID, req.TimeRange)
	if err != nil {
		return nil, fmt.Errorf("failed to get experiment summary: %w", err)
	}

	// Get metric events for all requested metrics
	metricResults := make(map[string]*MetricAnalysis)
	statisticalTests := make(map[string]*StatisticalTestResult)

	for _, metricName := range req.MetricNames {
		metricReq := &repository.MetricEventsRequest{
			MetricNames:   []string{metricName},
			ExperimentID:  req.ExperimentID,
			EnvironmentID: req.EnvironmentID,
			TimeRange:     req.TimeRange,
			Filters:       req.Filters,
		}

		events, err := s.eventRepo.GetMetricEvents(ctx, metricReq)
		if err != nil {
			log.Error().Err(err).Str("metric", metricName).Msg("Failed to get metric events")
			continue
		}

		// Analyze metric by variation
		analysis, err := s.analyzeMetricByVariation(events, metricName, req.Confidence)
		if err != nil {
			log.Error().Err(err).Str("metric", metricName).Msg("Failed to analyze metric")
			continue
		}

		metricResults[metricName] = analysis

		// Run statistical tests between variations
		if len(analysis.ByVariation) >= 2 {
			testResult, err := s.runVariationComparison(events, metricName)
			if err != nil {
				log.Error().Err(err).Str("metric", metricName).Msg("Failed to run statistical test")
			} else {
				statisticalTests[metricName] = testResult
			}
		}
	}

	// Generate variation results
	variationResults := make(map[string]*VariationResults)
	for _, variation := range summary.Variations {
		varResult := &VariationResults{
			VariationID:     variation.VariationID,
			ExposureCount:   variation.Exposures,
			UniqueUsers:     variation.UniqueUsers,
			ConversionRate:  variation.ConversionRate,
			AllocationRatio: variation.AllocationRatio,
			MetricSummaries: make(map[string]*DescriptiveStats),
		}

		// Add metric summaries for this variation
		for metricName, analysis := range metricResults {
			if stats, exists := analysis.ByVariation[variation.VariationID]; exists {
				varResult.MetricSummaries[metricName] = stats
			}
		}

		variationResults[variation.VariationID] = varResult
	}

	// Generate recommendations
	recommendations := s.generateRecommendations(summary, metricResults, statisticalTests)

	results := &ExperimentResults{
		ExperimentID:     req.ExperimentID,
		Summary:          summary,
		MetricResults:    metricResults,
		VariationResults: variationResults,
		StatisticalTests: statisticalTests,
		Recommendations:  recommendations,
		TimeRange:        req.TimeRange,
		LastUpdated:      time.Now(),
	}

	log.Info().
		Str("experiment_id", req.ExperimentID).
		Int("metric_count", len(metricResults)).
		Int("variation_count", len(variationResults)).
		Msg("Experiment results generated")

	return results, nil
}

func (s *experimentService) GetExperimentSummary(ctx context.Context, experimentID string, timeRange repository.TimeRange) (*repository.ExperimentSummary, error) {
	return s.eventRepo.GetExperimentSummary(ctx, experimentID, timeRange)
}

func (s *experimentService) GetExperimentTimeline(ctx context.Context, req *ExperimentTimelineRequest) (*ExperimentTimeline, error) {
	// TODO: Implement timeline analysis
	// This would involve:
	// 1. Segmenting the time range by granularity
	// 2. Calculating metrics for each time point
	// 3. Running statistical tests over time
	// 4. Detecting trends and change points
	// 5. Analyzing seasonality patterns

	log.Info().
		Str("experiment_id", req.ExperimentID).
		Str("granularity", req.Granularity).
		Msg("Timeline analysis requested - placeholder implementation")

	return &ExperimentTimeline{
		ExperimentID: req.ExperimentID,
		Granularity:  req.Granularity,
		TimePoints:   []TimePointData{},
		Trends:       make(map[string]*TrendAnalysis),
	}, nil
}

func (s *experimentService) AnalyzeExperiment(ctx context.Context, req *ExperimentAnalysisRequest) (*ExperimentAnalysis, error) {
	log.Info().
		Str("experiment_id", req.ExperimentID).
		Str("primary_metric", req.PrimaryMetric).
		Int("secondary_metrics", len(req.SecondaryMetrics)).
		Bool("sequential", req.Sequential).
		Bool("use_cuped", req.UseCUPED).
		Msg("Running comprehensive experiment analysis")

	// Set defaults
	if req.Alpha == 0 {
		req.Alpha = 0.05
	}
	if req.Power == 0 {
		req.Power = 0.8
	}

	// Analyze primary metric
	primaryResults, err := s.analyzeMetricComprehensive(ctx, req, req.PrimaryMetric)
	if err != nil {
		return nil, fmt.Errorf("failed to analyze primary metric: %w", err)
	}

	// Analyze secondary metrics
	secondaryResults := make(map[string]*MetricAnalysis)
	for _, metricName := range req.SecondaryMetrics {
		result, err := s.analyzeMetricComprehensive(ctx, req, metricName)
		if err != nil {
			log.Error().Err(err).Str("metric", metricName).Msg("Failed to analyze secondary metric")
			continue
		}
		secondaryResults[metricName] = result
	}

	// Generate overall conclusion
	conclusion := s.generateExperimentConclusion(primaryResults, secondaryResults, req.Alpha)

	// Run quality checks
	qualityChecks := s.runQualityAssessment(ctx, req)

	analysis := &ExperimentAnalysis{
		ExperimentID:      req.ExperimentID,
		AnalysisType:      s.getAnalysisType(req),
		PrimaryResults:    primaryResults,
		SecondaryResults:  secondaryResults,
		OverallConclusion: conclusion,
		QualityChecks:     qualityChecks,
		AnalysisTimestamp: time.Now(),
	}

	log.Info().
		Str("experiment_id", req.ExperimentID).
		Str("winner", conclusion.Winner).
		Float64("confidence", conclusion.Confidence).
		Str("recommendation", conclusion.Recommendation).
		Msg("Experiment analysis completed")

	return analysis, nil
}

// Helper methods
func (s *experimentService) analyzeMetricByVariation(events []repository.MetricEvent, metricName string, confidence float64) (*MetricAnalysis, error) {
	// Group events by variation
	variationData := make(map[string][]float64)
	allValues := []float64{}

	for _, event := range events {
		if event.MetricName == metricName {
			variationData[event.VariationID] = append(variationData[event.VariationID], event.Value)
			allValues = append(allValues, event.Value)
		}
	}

	// Calculate overall statistics
	overallStats, err := s.statsService.CalculateDescriptiveStats(allValues)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate overall stats: %w", err)
	}

	// Calculate stats by variation
	byVariation := make(map[string]*DescriptiveStats)
	tTestResults := make(map[string]*TTestResult)
	confidenceIntervals := make(map[string]*ConfidenceInterval)
	effectSizes := make(map[string]float64)

	// Find control variation (typically the one with most data or "control" name)
	controlVariation := s.findControlVariation(variationData)
	controlData := variationData[controlVariation]

	for variationID, data := range variationData {
		if len(data) == 0 {
			continue
		}

		// Calculate descriptive statistics
		stats, err := s.statsService.CalculateDescriptiveStats(data)
		if err != nil {
			log.Error().Err(err).Str("variation", variationID).Msg("Failed to calculate variation stats")
			continue
		}
		byVariation[variationID] = stats

		// Calculate confidence interval
		ci, err := s.statsService.CalculateConfidenceInterval(data, confidence)
		if err != nil {
			log.Error().Err(err).Str("variation", variationID).Msg("Failed to calculate confidence interval")
		} else {
			confidenceIntervals[variationID] = ci
		}

		// Run t-test against control (if not control itself)
		if variationID != controlVariation && len(controlData) > 0 {
			tTestReq := &TTestRequest{
				TreatmentData: data,
				ControlData:   controlData,
				Alpha:         0.05,
				Alternative:   "two-sided",
				EqualVar:      false, // Use Welch's t-test
			}

			tTestResult, err := s.statsService.RunTTest(tTestReq)
			if err != nil {
				log.Error().Err(err).Str("variation", variationID).Msg("Failed to run t-test")
			} else {
				tTestResults[variationID] = tTestResult
				effectSizes[variationID] = tTestResult.EffectSize
			}
		}
	}

	analysis := &MetricAnalysis{
		MetricName:          metricName,
		OverallStats:        overallStats,
		ByVariation:         byVariation,
		TTestResults:        tTestResults,
		ConfidenceIntervals: confidenceIntervals,
		EffectSizes:         effectSizes,
	}

	return analysis, nil
}

func (s *experimentService) runVariationComparison(events []repository.MetricEvent, metricName string) (*StatisticalTestResult, error) {
	// Get the two largest variations for comparison
	variationCounts := make(map[string]int)
	variationData := make(map[string][]float64)

	for _, event := range events {
		if event.MetricName == metricName {
			variationCounts[event.VariationID]++
			variationData[event.VariationID] = append(variationData[event.VariationID], event.Value)
		}
	}

	// Sort variations by count
	type variationCount struct {
		ID    string
		Count int
	}

	var sortedVariations []variationCount
	for id, count := range variationCounts {
		sortedVariations = append(sortedVariations, variationCount{ID: id, Count: count})
	}

	sort.Slice(sortedVariations, func(i, j int) bool {
		return sortedVariations[i].Count > sortedVariations[j].Count
	})

	if len(sortedVariations) < 2 {
		return nil, fmt.Errorf("insufficient variations for comparison")
	}

	// Run t-test between top two variations
	treatmentData := variationData[sortedVariations[0].ID]
	controlData := variationData[sortedVariations[1].ID]

	tTestReq := &TTestRequest{
		TreatmentData: treatmentData,
		ControlData:   controlData,
		Alpha:         0.05,
		Alternative:   "two-sided",
		EqualVar:      false,
	}

	result, err := s.statsService.RunTTest(tTestReq)
	if err != nil {
		return nil, err
	}

	return &StatisticalTestResult{
		TestType:      "t_test",
		PValue:        result.PValue,
		IsSignificant: result.IsSignificant,
		EffectSize:    result.EffectSize,
		Confidence:    0.95,
		Details:       result,
	}, nil
}

func (s *experimentService) findControlVariation(variationData map[string][]float64) string {
	// Heuristics to find control variation:
	// 1. Look for "control" in the name
	// 2. Use the variation with most data
	// 3. Use alphabetically first

	var controlVariation string
	maxDataPoints := 0

	for variationID, data := range variationData {
		// Check for "control" in name
		if variationID == "control" || variationID == "Control" || variationID == "CONTROL" {
			return variationID
		}

		// Track variation with most data
		if len(data) > maxDataPoints {
			maxDataPoints = len(data)
			controlVariation = variationID
		}
	}

	return controlVariation
}

func (s *experimentService) generateRecommendations(summary *repository.ExperimentSummary, metricResults map[string]*MetricAnalysis, statisticalTests map[string]*StatisticalTestResult) *ExperimentRecommendations {
	// TODO: Implement sophisticated recommendation engine
	// This would consider:
	// 1. Statistical significance across multiple metrics
	// 2. Effect sizes and practical significance
	// 3. Business context and cost considerations
	// 4. Risk assessment based on data quality
	// 5. Sample size adequacy

	reasoning := []string{"Placeholder recommendation logic"}
	nextSteps := []string{"Continue monitoring", "Prepare for launch decision"}

	return &ExperimentRecommendations{
		Decision:       "iterate",
		Confidence:     "medium",
		Reasoning:      reasoning,
		RiskAssessment: "moderate",
		NextSteps:      nextSteps,
		EstimatedImpact: &ImpactEstimate{
			RelativeLift:    0.05,
			AbsoluteImpact:  1000,
			ConfidenceRange: [2]float64{-0.02, 0.12},
		},
	}
}

func (s *experimentService) analyzeMetricComprehensive(ctx context.Context, req *ExperimentAnalysisRequest, metricName string) (*MetricAnalysis, error) {
	// Get metric events
	metricReq := &repository.MetricEventsRequest{
		MetricNames:   []string{metricName},
		ExperimentID:  req.ExperimentID,
		EnvironmentID: req.EnvironmentID,
		TimeRange:     req.TimeRange,
	}

	events, err := s.eventRepo.GetMetricEvents(ctx, metricReq)
	if err != nil {
		return nil, fmt.Errorf("failed to get metric events: %w", err)
	}

	// Perform basic analysis
	analysis, err := s.analyzeMetricByVariation(events, metricName, 0.95)
	if err != nil {
		return nil, err
	}

	// Apply CUPED if requested
	if req.UseCUPED && req.CovariateMetric != "" {
		// TODO: Implement CUPED analysis
		log.Info().
			Str("metric", metricName).
			Str("covariate", req.CovariateMetric).
			Msg("CUPED analysis requested - placeholder implementation")
	}

	// Run sequential analysis if requested
	if req.Sequential {
		// TODO: Implement sequential analysis
		log.Info().
			Str("metric", metricName).
			Msg("Sequential analysis requested - placeholder implementation")
	}

	return analysis, nil
}

func (s *experimentService) generateExperimentConclusion(primaryResults *MetricAnalysis, secondaryResults map[string]*MetricAnalysis, alpha float64) *ExperimentConclusion {
	// TODO: Implement comprehensive conclusion generation
	// This would consider:
	// 1. Primary metric significance and effect size
	// 2. Secondary metric alignment
	// 3. Multiple testing corrections
	// 4. Business impact estimates

	return &ExperimentConclusion{
		Winner:         "inconclusive",
		Confidence:     0.5,
		ExpectedLift:   0.0,
		RiskAssessment: "moderate",
		Recommendation: "continue_monitoring",
		BusinessImpact: &BusinessImpact{
			EstimatedLift:       0.0,
			LiftConfidenceRange: [2]float64{-0.05, 0.05},
		},
	}
}

func (s *experimentService) runQualityAssessment(ctx context.Context, req *ExperimentAnalysisRequest) *QualityAssessment {
	// TODO: Implement quality assessment checks
	// This would include:
	// 1. Sample size adequacy checks
	// 2. Balance testing across variations
	// 3. Data quality validation
	// 4. Bias detection
	// 5. External validity assessment

	return &QualityAssessment{
		SampleSizeAdequacy:   "adequate",
		BalanceCheck:         "balanced",
		DataQualityIssues:    []string{},
		BiasAssessment:       "low_risk",
		ExternalValidityRisk: "moderate",
	}
}

func (s *experimentService) getAnalysisType(req *ExperimentAnalysisRequest) string {
	if req.Sequential {
		return "sequential"
	}
	return "frequentist"
}
