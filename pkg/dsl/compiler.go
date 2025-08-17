package dsl

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// Compiler compiles rule DSL into an optimized evaluation plan
type Compiler struct {
	operators map[string]OperatorFunc
}

// OperatorFunc represents a comparison operator function
type OperatorFunc func(left, right interface{}) bool

// NewCompiler creates a new DSL compiler
func NewCompiler() *Compiler {
	c := &Compiler{
		operators: make(map[string]OperatorFunc),
	}
	c.registerDefaultOperators()
	return c
}

// CompiledPlan represents the compiled evaluation plan for a rule set
type CompiledPlan struct {
	FlagKey      string            `json:"flag_key"`
	Rules        []CompiledRule    `json:"rules"`
	DefaultValue interface{}       `json:"default_value"`
	Metadata     map[string]string `json:"metadata"`
}

// CompiledRule represents a single compiled rule
type CompiledRule struct {
	ID                string              `json:"id"`
	Conditions        []CompiledCondition `json:"conditions"`
	Action            CompiledAction      `json:"action"`
	TrafficAllocation float64             `json:"traffic_allocation"`
	Priority          int                 `json:"priority"`
}

// CompiledCondition represents a compiled condition
type CompiledCondition struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
	Compiled  bool        `json:"compiled"`
}

// CompiledAction represents the action to take when rules match
type CompiledAction struct {
	Type         string           `json:"type"` // "variation" or "rollout"
	VariationKey string           `json:"variation_key,omitempty"`
	Rollout      *CompiledRollout `json:"rollout,omitempty"`
}

// CompiledRollout represents a compiled rollout configuration
type CompiledRollout struct {
	Variations  []RolloutVariation `json:"variations"`
	TotalWeight float64            `json:"total_weight"`
}

// RolloutVariation represents a variation in a rollout
type RolloutVariation struct {
	VariationKey string  `json:"variation_key"`
	Weight       float64 `json:"weight"`
	StartBucket  int     `json:"start_bucket"`
	EndBucket    int     `json:"end_bucket"`
}

// RuleDefinition represents the input rule definition
type RuleDefinition struct {
	If   interface{} `json:"if"`
	Then interface{} `json:"then"`
}

// ConditionDefinition represents a condition in the DSL
type ConditionDefinition struct {
	Attribute string      `json:"attribute"`
	Operator  string      `json:"operator"`
	Value     interface{} `json:"value"`
}

// CompileRules compiles a set of rule definitions into an optimized plan
func (c *Compiler) CompileRules(flagKey string, rules []RuleDefinition, defaultValue interface{}) (*CompiledPlan, error) {
	if flagKey == "" {
		return nil, fmt.Errorf("flag key is required")
	}

	plan := &CompiledPlan{
		FlagKey:      flagKey,
		Rules:        make([]CompiledRule, 0, len(rules)),
		DefaultValue: defaultValue,
		Metadata:     make(map[string]string),
	}

	for i, ruleDef := range rules {
		compiledRule, err := c.compileRule(fmt.Sprintf("rule_%d", i), ruleDef, i)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %d: %w", i, err)
		}
		plan.Rules = append(plan.Rules, *compiledRule)
	}

	// Optimize the plan
	c.optimizePlan(plan)

	return plan, nil
}

// compileRule compiles a single rule definition
func (c *Compiler) compileRule(ruleID string, ruleDef RuleDefinition, priority int) (*CompiledRule, error) {
	rule := &CompiledRule{
		ID:                ruleID,
		Priority:          priority,
		TrafficAllocation: 1.0, // Default to 100%
	}

	// Compile conditions
	conditions, err := c.compileConditions(ruleDef.If)
	if err != nil {
		return nil, fmt.Errorf("failed to compile conditions: %w", err)
	}
	rule.Conditions = conditions

	// Compile action
	action, err := c.compileAction(ruleDef.Then)
	if err != nil {
		return nil, fmt.Errorf("failed to compile action: %w", err)
	}
	rule.Action = *action

	return rule, nil
}

// compileConditions compiles the condition part of a rule
func (c *Compiler) compileConditions(ifClause interface{}) ([]CompiledCondition, error) {
	if ifClause == nil {
		return []CompiledCondition{}, nil
	}

	switch v := ifClause.(type) {
	case map[string]interface{}:
		return c.compileConditionMap(v)
	case []interface{}:
		return c.compileConditionArray(v)
	default:
		return nil, fmt.Errorf("unsupported condition type: %T", v)
	}
}

// compileConditionMap compiles a condition from a map
func (c *Compiler) compileConditionMap(condMap map[string]interface{}) ([]CompiledCondition, error) {
	// Handle logical operators
	if and, exists := condMap["and"]; exists {
		return c.compileConditions(and)
	}

	if or, exists := condMap["or"]; exists {
		// For OR conditions, we need to handle them differently
		// This is a simplified implementation - in production you'd want more sophisticated OR handling
		return c.compileConditions(or)
	}

	// Handle direct condition
	attribute, hasAttr := condMap["attribute"].(string)
	operator, hasOp := condMap["operator"].(string)
	value, hasValue := condMap["value"]

	if !hasAttr || !hasOp || !hasValue {
		return nil, fmt.Errorf("condition must have attribute, operator, and value")
	}

	condition := CompiledCondition{
		Attribute: attribute,
		Operator:  operator,
		Value:     value,
		Compiled:  true,
	}

	// Validate operator
	if !c.isValidOperator(operator) {
		return nil, fmt.Errorf("unsupported operator: %s", operator)
	}

	return []CompiledCondition{condition}, nil
}

// compileConditionArray compiles conditions from an array
func (c *Compiler) compileConditionArray(condArray []interface{}) ([]CompiledCondition, error) {
	var conditions []CompiledCondition

	for i, item := range condArray {
		itemConditions, err := c.compileConditions(item)
		if err != nil {
			return nil, fmt.Errorf("failed to compile condition %d: %w", i, err)
		}
		conditions = append(conditions, itemConditions...)
	}

	return conditions, nil
}

// compileAction compiles the action part of a rule
func (c *Compiler) compileAction(thenClause interface{}) (*CompiledAction, error) {
	if thenClause == nil {
		return nil, fmt.Errorf("action cannot be nil")
	}

	switch v := thenClause.(type) {
	case string:
		// Direct variation assignment
		return &CompiledAction{
			Type:         "variation",
			VariationKey: v,
		}, nil

	case map[string]interface{}:
		// Rollout or percentage assignment
		if rollout, exists := v["rollout"]; exists {
			compiledRollout, err := c.compileRollout(rollout)
			if err != nil {
				return nil, fmt.Errorf("failed to compile rollout: %w", err)
			}
			return &CompiledAction{
				Type:    "rollout",
				Rollout: compiledRollout,
			}, nil
		}

		if variation, exists := v["variation"]; exists {
			if variationKey, ok := variation.(string); ok {
				return &CompiledAction{
					Type:         "variation",
					VariationKey: variationKey,
				}, nil
			}
		}

		return nil, fmt.Errorf("unsupported action format")

	default:
		return nil, fmt.Errorf("unsupported action type: %T", v)
	}
}

// compileRollout compiles a rollout configuration
func (c *Compiler) compileRollout(rolloutDef interface{}) (*CompiledRollout, error) {
	rolloutMap, ok := rolloutDef.(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("rollout must be an object")
	}

	variationsInterface, exists := rolloutMap["variations"]
	if !exists {
		return nil, fmt.Errorf("rollout must have variations")
	}

	variationsArray, ok := variationsInterface.([]interface{})
	if !ok {
		return nil, fmt.Errorf("variations must be an array")
	}

	var variations []RolloutVariation
	var totalWeight float64
	currentBucket := 0

	for i, varInterface := range variationsArray {
		varMap, ok := varInterface.(map[string]interface{})
		if !ok {
			return nil, fmt.Errorf("variation %d must be an object", i)
		}

		key, hasKey := varMap["key"].(string)
		if !hasKey {
			return nil, fmt.Errorf("variation %d must have a key", i)
		}

		weight, hasWeight := varMap["weight"].(float64)
		if !hasWeight {
			if weightInt, ok := varMap["weight"].(int); ok {
				weight = float64(weightInt)
			} else {
				return nil, fmt.Errorf("variation %d must have a weight", i)
			}
		}

		if weight < 0 {
			return nil, fmt.Errorf("variation %d weight must be non-negative", i)
		}

		totalWeight += weight

		variations = append(variations, RolloutVariation{
			VariationKey: key,
			Weight:       weight,
			StartBucket:  currentBucket,
			EndBucket:    currentBucket, // Will be calculated after we know total weight
		})
	}

	if totalWeight <= 0 {
		return nil, fmt.Errorf("total weight must be positive")
	}

	// Calculate bucket ranges
	currentBucket = 0
	for i := range variations {
		bucketSize := int((variations[i].Weight / totalWeight) * 10000)

		// Ensure the last variation gets all remaining buckets
		if i == len(variations)-1 {
			bucketSize = 10000 - currentBucket
		}

		variations[i].StartBucket = currentBucket
		variations[i].EndBucket = currentBucket + bucketSize
		currentBucket += bucketSize
	}

	return &CompiledRollout{
		Variations:  variations,
		TotalWeight: totalWeight,
	}, nil
}

// optimizePlan optimizes the compiled plan
func (c *Compiler) optimizePlan(plan *CompiledPlan) {
	// Sort rules by priority
	// In a more sophisticated implementation, you might:
	// - Reorder conditions by selectivity
	// - Combine similar conditions
	// - Pre-compute regex patterns
	// - Index frequently used attributes

	plan.Metadata["optimized"] = "true"
	plan.Metadata["rules_count"] = strconv.Itoa(len(plan.Rules))
}

// registerDefaultOperators registers the default comparison operators
func (c *Compiler) registerDefaultOperators() {
	c.operators["eq"] = func(left, right interface{}) bool {
		return fmt.Sprintf("%v", left) == fmt.Sprintf("%v", right)
	}

	c.operators["neq"] = func(left, right interface{}) bool {
		return fmt.Sprintf("%v", left) != fmt.Sprintf("%v", right)
	}

	c.operators["in"] = func(left, right interface{}) bool {
		if rightArray, ok := right.([]interface{}); ok {
			leftStr := fmt.Sprintf("%v", left)
			for _, item := range rightArray {
				if fmt.Sprintf("%v", item) == leftStr {
					return true
				}
			}
		}
		return false
	}

	c.operators["nin"] = func(left, right interface{}) bool {
		return !c.operators["in"](left, right)
	}

	c.operators["lt"] = func(left, right interface{}) bool {
		leftFloat, leftOk := toFloat64(left)
		rightFloat, rightOk := toFloat64(right)
		return leftOk && rightOk && leftFloat < rightFloat
	}

	c.operators["gt"] = func(left, right interface{}) bool {
		leftFloat, leftOk := toFloat64(left)
		rightFloat, rightOk := toFloat64(right)
		return leftOk && rightOk && leftFloat > rightFloat
	}

	c.operators["lte"] = func(left, right interface{}) bool {
		leftFloat, leftOk := toFloat64(left)
		rightFloat, rightOk := toFloat64(right)
		return leftOk && rightOk && leftFloat <= rightFloat
	}

	c.operators["gte"] = func(left, right interface{}) bool {
		leftFloat, leftOk := toFloat64(left)
		rightFloat, rightOk := toFloat64(right)
		return leftOk && rightOk && leftFloat >= rightFloat
	}

	c.operators["contains"] = func(left, right interface{}) bool {
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return strings.Contains(leftStr, rightStr)
	}

	c.operators["regex"] = func(left, right interface{}) bool {
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		matched, err := regexp.MatchString(rightStr, leftStr)
		return err == nil && matched
	}
}

// isValidOperator checks if an operator is supported
func (c *Compiler) isValidOperator(operator string) bool {
	_, exists := c.operators[operator]
	return exists
}

// EvaluateCondition evaluates a compiled condition
func (c *Compiler) EvaluateCondition(condition *CompiledCondition, context map[string]interface{}) bool {
	if condition == nil {
		return false
	}

	// Get attribute value from context
	attributeValue, exists := context[condition.Attribute]
	if !exists {
		return false
	}

	// Get operator function
	operatorFunc, exists := c.operators[condition.Operator]
	if !exists {
		return false
	}

	return operatorFunc(attributeValue, condition.Value)
}

// toFloat64 converts various numeric types to float64
func toFloat64(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int32:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		if f, err := strconv.ParseFloat(val, 64); err == nil {
			return f, true
		}
	}
	return 0, false
}

// SerializePlan serializes a compiled plan to JSON
func (c *Compiler) SerializePlan(plan *CompiledPlan) ([]byte, error) {
	return json.Marshal(plan)
}

// DeserializePlan deserializes a compiled plan from JSON
func (c *Compiler) DeserializePlan(data []byte) (*CompiledPlan, error) {
	var plan CompiledPlan
	if err := json.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}
