package services

import (
	"fmt"
	"math"
	"sort"

	"github.com/rs/zerolog/log"
)

type StatisticsService interface {
	RunTTest(req *TTestRequest) (*TTestResult, error)
	RunChiSquareTest(req *ChiSquareRequest) (*ChiSquareResult, error)
	RunSequentialAnalysis(req *SequentialAnalysisRequest) (*SequentialAnalysisResult, error)
	CalculateConfidenceInterval(data []float64, confidence float64) (*ConfidenceInterval, error)
	CalculateDescriptiveStats(data []float64) (*DescriptiveStats, error)
	ApplyCUPED(treatment, control, covariate []float64) (*CUPEDResult, error)
}

type statisticsService struct{}

func NewStatisticsService() StatisticsService {
	return &statisticsService{}
}

// Request types
type TTestRequest struct {
	TreatmentData []float64 `json:"treatment_data"`
	ControlData   []float64 `json:"control_data"`
	Alpha         float64   `json:"alpha"`       // Significance level (default: 0.05)
	Alternative   string    `json:"alternative"` // "two-sided", "greater", "less"
	EqualVar      bool      `json:"equal_var"`   // Assume equal variances (default: false - Welch's t-test)
}

type ChiSquareRequest struct {
	ObservedFrequencies [][]int `json:"observed_frequencies"` // 2D array for contingency table
	Alpha               float64 `json:"alpha"`                // Significance level (default: 0.05)
}

type SequentialAnalysisRequest struct {
	TreatmentData    []float64 `json:"treatment_data"`
	ControlData      []float64 `json:"control_data"`
	Alpha            float64   `json:"alpha"`             // Overall Type I error rate
	Power            float64   `json:"power"`             // Desired power (1 - Type II error rate)
	MinEffect        float64   `json:"min_effect"`        // Minimum detectable effect size
	SpendingFunction string    `json:"spending_function"` // "pocock", "obrien_fleming", "alpha_spending"
	MaxAnalyses      int       `json:"max_analyses"`      // Maximum number of interim analyses
	CurrentAnalysis  int       `json:"current_analysis"`  // Current analysis number (1-indexed)
}

// Result types
type TTestResult struct {
	Statistic        float64           `json:"statistic"`
	PValue           float64           `json:"p_value"`
	DegreesOfFreedom float64           `json:"degrees_of_freedom"`
	CriticalValue    float64           `json:"critical_value"`
	IsSignificant    bool              `json:"is_significant"`
	EffectSize       float64           `json:"effect_size"` // Cohen's d
	PowerAnalysis    *PowerAnalysis    `json:"power_analysis,omitempty"`
	TreatmentStats   *DescriptiveStats `json:"treatment_stats"`
	ControlStats     *DescriptiveStats `json:"control_stats"`
}

type ChiSquareResult struct {
	Statistic           float64     `json:"statistic"`
	PValue              float64     `json:"p_value"`
	DegreesOfFreedom    int         `json:"degrees_of_freedom"`
	CriticalValue       float64     `json:"critical_value"`
	IsSignificant       bool        `json:"is_significant"`
	ExpectedFrequencies [][]float64 `json:"expected_frequencies"`
	Residuals           [][]float64 `json:"residuals"`
	CramerV             float64     `json:"cramer_v"` // Effect size measure
}

type SequentialAnalysisResult struct {
	CurrentBoundary float64 `json:"current_boundary"`
	TestStatistic   float64 `json:"test_statistic"`
	IsSignificant   bool    `json:"is_significant"`
	ShouldStop      bool    `json:"should_stop"`
	SpentAlpha      float64 `json:"spent_alpha"`
	RemainingAlpha  float64 `json:"remaining_alpha"`
	EstimatedEffect float64 `json:"estimated_effect"`
	ConditionPower  float64 `json:"conditional_power"`
	PredictivePower float64 `json:"predictive_power"`
}

type ConfidenceInterval struct {
	Lower      float64 `json:"lower"`
	Upper      float64 `json:"upper"`
	Confidence float64 `json:"confidence"`
}

type DescriptiveStats struct {
	Count        int     `json:"count"`
	Mean         float64 `json:"mean"`
	Median       float64 `json:"median"`
	StdDev       float64 `json:"std_dev"`
	Variance     float64 `json:"variance"`
	Min          float64 `json:"min"`
	Max          float64 `json:"max"`
	Q1           float64 `json:"q1"`
	Q3           float64 `json:"q3"`
	IQR          float64 `json:"iqr"`
	Skewness     float64 `json:"skewness"`
	Kurtosis     float64 `json:"kurtosis"`
	Percentile95 float64 `json:"percentile_95"`
	Percentile99 float64 `json:"percentile_99"`
}

type PowerAnalysis struct {
	ObservedPower       float64 `json:"observed_power"`
	RequiredSampleSize  int     `json:"required_sample_size"`
	MinDetectableEffect float64 `json:"min_detectable_effect"`
}

type CUPEDResult struct {
	AdjustedTreatmentData []float64 `json:"adjusted_treatment_data"`
	AdjustedControlData   []float64 `json:"adjusted_control_data"`
	VarianceReduction     float64   `json:"variance_reduction"`
	ThetaCoefficient      float64   `json:"theta_coefficient"`
}

// Implementation
func (s *statisticsService) RunTTest(req *TTestRequest) (*TTestResult, error) {
	if len(req.TreatmentData) == 0 || len(req.ControlData) == 0 {
		return nil, fmt.Errorf("both treatment and control data must be non-empty")
	}

	// Set defaults
	if req.Alpha == 0 {
		req.Alpha = 0.05
	}
	if req.Alternative == "" {
		req.Alternative = "two-sided"
	}

	// Calculate descriptive statistics
	treatmentStats, err := s.CalculateDescriptiveStats(req.TreatmentData)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate treatment stats: %w", err)
	}

	controlStats, err := s.CalculateDescriptiveStats(req.ControlData)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate control stats: %w", err)
	}

	// Calculate t-statistic and degrees of freedom
	var tStat, df float64
	if req.EqualVar {
		// Student's t-test (equal variances)
		tStat, df = s.studentTTest(req.TreatmentData, req.ControlData, treatmentStats, controlStats)
	} else {
		// Welch's t-test (unequal variances) - default
		tStat, df = s.welchTTest(req.TreatmentData, req.ControlData, treatmentStats, controlStats)
	}

	// Calculate p-value
	pValue := s.calculateTTestPValue(tStat, df, req.Alternative)

	// Calculate effect size (Cohen's d)
	pooledStdDev := math.Sqrt(((float64(len(req.TreatmentData))-1)*treatmentStats.Variance + (float64(len(req.ControlData))-1)*controlStats.Variance) / (float64(len(req.TreatmentData)) + float64(len(req.ControlData)) - 2))
	effectSize := (treatmentStats.Mean - controlStats.Mean) / pooledStdDev

	// Determine significance
	isSignificant := pValue < req.Alpha

	// Calculate critical value (for two-tailed test)
	criticalValue := s.tCriticalValue(df, req.Alpha/2)

	result := &TTestResult{
		Statistic:        tStat,
		PValue:           pValue,
		DegreesOfFreedom: df,
		CriticalValue:    criticalValue,
		IsSignificant:    isSignificant,
		EffectSize:       effectSize,
		TreatmentStats:   treatmentStats,
		ControlStats:     controlStats,
	}

	log.Debug().
		Float64("t_statistic", tStat).
		Float64("p_value", pValue).
		Float64("effect_size", effectSize).
		Bool("significant", isSignificant).
		Msg("T-test completed")

	return result, nil
}

func (s *statisticsService) RunChiSquareTest(req *ChiSquareRequest) (*ChiSquareResult, error) {
	if len(req.ObservedFrequencies) == 0 || len(req.ObservedFrequencies[0]) == 0 {
		return nil, fmt.Errorf("observed frequencies must be non-empty")
	}

	if req.Alpha == 0 {
		req.Alpha = 0.05
	}

	rows := len(req.ObservedFrequencies)
	cols := len(req.ObservedFrequencies[0])

	// Calculate row and column totals
	rowTotals := make([]int, rows)
	colTotals := make([]int, cols)
	grandTotal := 0

	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			rowTotals[i] += req.ObservedFrequencies[i][j]
			colTotals[j] += req.ObservedFrequencies[i][j]
			grandTotal += req.ObservedFrequencies[i][j]
		}
	}

	// Calculate expected frequencies
	expectedFreq := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		expectedFreq[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			expectedFreq[i][j] = float64(rowTotals[i]*colTotals[j]) / float64(grandTotal)
		}
	}

	// Calculate chi-square statistic
	chiSquare := 0.0
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			observed := float64(req.ObservedFrequencies[i][j])
			expected := expectedFreq[i][j]
			if expected > 0 {
				chiSquare += math.Pow(observed-expected, 2) / expected
			}
		}
	}

	// Degrees of freedom
	df := (rows - 1) * (cols - 1)

	// Calculate p-value (using chi-square distribution)
	pValue := s.chiSquarePValue(chiSquare, df)

	// Critical value
	criticalValue := s.chiSquareCriticalValue(df, req.Alpha)

	// Calculate Cramer's V (effect size)
	cramerV := math.Sqrt(chiSquare / (float64(grandTotal) * float64(min(rows-1, cols-1))))

	// Calculate residuals
	residuals := make([][]float64, rows)
	for i := 0; i < rows; i++ {
		residuals[i] = make([]float64, cols)
		for j := 0; j < cols; j++ {
			observed := float64(req.ObservedFrequencies[i][j])
			expected := expectedFreq[i][j]
			if expected > 0 {
				residuals[i][j] = (observed - expected) / math.Sqrt(expected)
			}
		}
	}

	result := &ChiSquareResult{
		Statistic:           chiSquare,
		PValue:              pValue,
		DegreesOfFreedom:    df,
		CriticalValue:       criticalValue,
		IsSignificant:       pValue < req.Alpha,
		ExpectedFrequencies: expectedFreq,
		Residuals:           residuals,
		CramerV:             cramerV,
	}

	log.Debug().
		Float64("chi_square", chiSquare).
		Float64("p_value", pValue).
		Float64("cramer_v", cramerV).
		Bool("significant", result.IsSignificant).
		Msg("Chi-square test completed")

	return result, nil
}

func (s *statisticsService) RunSequentialAnalysis(req *SequentialAnalysisRequest) (*SequentialAnalysisResult, error) {
	// TODO: Implement full sequential analysis with alpha spending functions
	// This is a complex implementation that would include:
	// 1. Alpha spending function calculations (Pocock, O'Brien-Fleming)
	// 2. Information fraction calculations
	// 3. Boundary calculations for interim analyses
	// 4. Conditional power calculations
	// 5. Predictive power calculations

	log.Info().
		Int("current_analysis", req.CurrentAnalysis).
		Int("max_analyses", req.MaxAnalyses).
		Str("spending_function", req.SpendingFunction).
		Msg("Sequential analysis requested - placeholder implementation")

	// Placeholder implementation
	return &SequentialAnalysisResult{
		CurrentBoundary: 1.96, // Standard normal boundary
		TestStatistic:   0.0,
		IsSignificant:   false,
		ShouldStop:      false,
		SpentAlpha:      0.01,
		RemainingAlpha:  req.Alpha - 0.01,
		EstimatedEffect: 0.0,
		ConditionPower:  0.5,
		PredictivePower: 0.3,
	}, nil
}

func (s *statisticsService) CalculateConfidenceInterval(data []float64, confidence float64) (*ConfidenceInterval, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data must be non-empty")
	}

	stats, err := s.CalculateDescriptiveStats(data)
	if err != nil {
		return nil, err
	}

	alpha := 1.0 - confidence
	df := float64(len(data) - 1)
	tCrit := s.tCriticalValue(df, alpha/2)

	margin := tCrit * (stats.StdDev / math.Sqrt(float64(len(data))))

	return &ConfidenceInterval{
		Lower:      stats.Mean - margin,
		Upper:      stats.Mean + margin,
		Confidence: confidence,
	}, nil
}

func (s *statisticsService) CalculateDescriptiveStats(data []float64) (*DescriptiveStats, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("data must be non-empty")
	}

	n := len(data)
	sorted := make([]float64, n)
	copy(sorted, data)
	sort.Float64s(sorted)

	// Basic statistics
	sum := 0.0
	for _, v := range data {
		sum += v
	}
	mean := sum / float64(n)

	// Variance and standard deviation
	sumSquaredDiffs := 0.0
	for _, v := range data {
		diff := v - mean
		sumSquaredDiffs += diff * diff
	}
	variance := sumSquaredDiffs / float64(n-1)
	stdDev := math.Sqrt(variance)

	// Percentiles
	median := s.percentile(sorted, 0.5)
	q1 := s.percentile(sorted, 0.25)
	q3 := s.percentile(sorted, 0.75)
	p95 := s.percentile(sorted, 0.95)
	p99 := s.percentile(sorted, 0.99)

	// Skewness and kurtosis
	skewness := s.calculateSkewness(data, mean, stdDev)
	kurtosis := s.calculateKurtosis(data, mean, stdDev)

	return &DescriptiveStats{
		Count:        n,
		Mean:         mean,
		Median:       median,
		StdDev:       stdDev,
		Variance:     variance,
		Min:          sorted[0],
		Max:          sorted[n-1],
		Q1:           q1,
		Q3:           q3,
		IQR:          q3 - q1,
		Skewness:     skewness,
		Kurtosis:     kurtosis,
		Percentile95: p95,
		Percentile99: p99,
	}, nil
}

func (s *statisticsService) ApplyCUPED(treatment, control, covariate []float64) (*CUPEDResult, error) {
	// TODO: Implement CUPED (Controlled-experiment Using Pre-Experiment Data)
	// This would involve:
	// 1. Calculate theta coefficient using covariate data
	// 2. Adjust treatment and control data using the covariate
	// 3. Calculate variance reduction achieved

	log.Info().
		Int("treatment_size", len(treatment)).
		Int("control_size", len(control)).
		Int("covariate_size", len(covariate)).
		Msg("CUPED analysis requested - placeholder implementation")

	return &CUPEDResult{
		AdjustedTreatmentData: treatment, // Placeholder
		AdjustedControlData:   control,   // Placeholder
		VarianceReduction:     0.15,      // Placeholder: 15% variance reduction
		ThetaCoefficient:      0.5,       // Placeholder
	}, nil
}

// Helper functions
func (s *statisticsService) welchTTest(treatment, control []float64, treatmentStats, controlStats *DescriptiveStats) (float64, float64) {
	// Welch's t-test for unequal variances
	meanDiff := treatmentStats.Mean - controlStats.Mean

	n1, n2 := float64(len(treatment)), float64(len(control))
	s1, s2 := treatmentStats.Variance, controlStats.Variance

	// Standard error
	se := math.Sqrt(s1/n1 + s2/n2)

	// t-statistic
	tStat := meanDiff / se

	// Welch-Satterthwaite degrees of freedom
	numerator := math.Pow(s1/n1+s2/n2, 2)
	denominator := math.Pow(s1/n1, 2)/(n1-1) + math.Pow(s2/n2, 2)/(n2-1)
	df := numerator / denominator

	return tStat, df
}

func (s *statisticsService) studentTTest(treatment, control []float64, treatmentStats, controlStats *DescriptiveStats) (float64, float64) {
	// Student's t-test for equal variances
	meanDiff := treatmentStats.Mean - controlStats.Mean

	n1, n2 := float64(len(treatment)), float64(len(control))
	s1, s2 := treatmentStats.Variance, controlStats.Variance

	// Pooled variance
	pooledVar := ((n1-1)*s1 + (n2-1)*s2) / (n1 + n2 - 2)

	// Standard error
	se := math.Sqrt(pooledVar * (1/n1 + 1/n2))

	// t-statistic
	tStat := meanDiff / se

	// Degrees of freedom
	df := n1 + n2 - 2

	return tStat, df
}

func (s *statisticsService) calculateTTestPValue(tStat, df float64, alternative string) float64 {
	// TODO: Implement proper t-distribution CDF
	// For now, using normal approximation for large df
	absTStat := math.Abs(tStat)

	switch alternative {
	case "two-sided":
		return 2 * s.normalCDF(-absTStat)
	case "greater":
		return s.normalCDF(-tStat)
	case "less":
		return s.normalCDF(tStat)
	default:
		return 2 * s.normalCDF(-absTStat)
	}
}

func (s *statisticsService) normalCDF(x float64) float64 {
	// Approximation of standard normal CDF
	return 0.5 * (1 + math.Erf(x/math.Sqrt2))
}

func (s *statisticsService) tCriticalValue(df, alpha float64) float64 {
	// TODO: Implement proper t-distribution quantile function
	// For now, using normal approximation
	if alpha == 0.025 {
		return 1.96 // 95% confidence
	}
	return 1.96 // Placeholder
}

func (s *statisticsService) chiSquarePValue(chiSquare float64, df int) float64 {
	// TODO: Implement proper chi-square distribution CDF
	// Placeholder implementation
	return 0.05
}

func (s *statisticsService) chiSquareCriticalValue(df int, alpha float64) float64 {
	// TODO: Implement proper chi-square distribution quantile function
	// Placeholder implementation
	return 3.84 // chi-square critical value for df=1, alpha=0.05
}

func (s *statisticsService) percentile(sorted []float64, p float64) float64 {
	n := len(sorted)
	index := p * float64(n-1)

	if index == float64(int(index)) {
		return sorted[int(index)]
	}

	lower := int(math.Floor(index))
	upper := int(math.Ceil(index))
	weight := index - float64(lower)

	return sorted[lower]*(1-weight) + sorted[upper]*weight
}

func (s *statisticsService) calculateSkewness(data []float64, mean, stdDev float64) float64 {
	n := float64(len(data))
	sum := 0.0

	for _, v := range data {
		sum += math.Pow((v-mean)/stdDev, 3)
	}

	return (n / ((n - 1) * (n - 2))) * sum
}

func (s *statisticsService) calculateKurtosis(data []float64, mean, stdDev float64) float64 {
	n := float64(len(data))
	sum := 0.0

	for _, v := range data {
		sum += math.Pow((v-mean)/stdDev, 4)
	}

	return ((n*(n+1))/((n-1)*(n-2)*(n-3)))*sum - (3*math.Pow(n-1, 2))/((n-2)*(n-3))
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
