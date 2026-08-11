// Package decision translates reliability evaluation results and active
// burn-rate alerts into explainable delivery recommendations.
package decision

import (
	"fmt"
	"sort"
	"time"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/evaluation"
)

// Recommendation is advisory guidance for a delivery system.
type Recommendation string

const (
	Proceed Recommendation = "proceed"
	Caution Recommendation = "caution"
	Pause   Recommendation = "pause"
	Unknown Recommendation = "unknown"
)

// Result is the stable, machine-readable decision interface.
type Result struct {
	APIVersion       string         `json:"apiVersion"`
	Kind             string         `json:"kind"`
	ServiceName      string         `json:"serviceName"`
	EvaluatedAt      time.Time      `json:"evaluatedAt"`
	Recommendation   Recommendation `json:"recommendation"`
	ActiveSeverities []string       `json:"activeSeverities"`
	Policy           Policy         `json:"policy"`
	Reasons          []Reason       `json:"reasons"`
}

// Policy exposes the thresholds that formed the recommendation.
type Policy struct {
	CautionBelowPercent   float64  `json:"cautionBelowPercent"`
	PauseAtOrBelowPercent float64  `json:"pauseAtOrBelowPercent"`
	PauseSeverities       []string `json:"pauseSeverities"`
}

// Reason makes each decision condition independently inspectable.
type Reason struct {
	Code        string   `json:"code"`
	Message     string   `json:"message"`
	ObjectiveID string   `json:"objectiveId,omitempty"`
	Severity    string   `json:"severity,omitempty"`
	Observed    *float64 `json:"observedPercent,omitempty"`
	Threshold   *float64 `json:"thresholdPercent,omitempty"`
}

// Decide applies the contract policy to an evaluation and active alert
// severities. Recommendation precedence is pause, unknown, caution, proceed.
func Decide(reliability *contract.ServiceReliability, result *evaluation.Result, activeSeverities []string) Result {
	policy := reliability.Spec.DecisionPolicy
	active := uniqueSorted(activeSeverities)
	pauseSeverities := uniqueSorted(policy.Pause.ActiveBurnRateSeverities)
	decision := Result{
		APIVersion:       "reliability.vaipex.io/v1alpha1",
		Kind:             "ReliabilityDecision",
		ServiceName:      result.ServiceName,
		EvaluatedAt:      result.EvaluatedAt,
		ActiveSeverities: active,
		Policy: Policy{
			CautionBelowPercent:   policy.Caution.RemainingBudgetBelowPercent,
			PauseAtOrBelowPercent: policy.Pause.RemainingBudgetAtOrBelowPercent,
			PauseSeverities:       pauseSeverities,
		},
		Reasons: make([]Reason, 0),
	}

	pauseSeveritySet := make(map[string]struct{}, len(pauseSeverities))
	for _, severity := range pauseSeverities {
		pauseSeveritySet[severity] = struct{}{}
	}
	hasPause := false
	hasUnknown := false
	hasCaution := false
	for _, severity := range active {
		if _, matched := pauseSeveritySet[severity]; !matched {
			continue
		}
		hasPause = true
		decision.Reasons = append(decision.Reasons, Reason{
			Code:     "active_burn_rate_severity",
			Severity: severity,
			Message:  fmt.Sprintf("active %s burn-rate alert matches the pause policy", severity),
		})
	}

	for _, objective := range result.Objectives {
		if objective.Status == "unknown" || objective.BudgetRemainingPercent == nil {
			hasUnknown = true
			message := "objective does not have usable reliability evidence"
			if objective.Reason != "" {
				message += ": " + objective.Reason
			}
			decision.Reasons = append(decision.Reasons, Reason{
				Code:        "unknown_evidence",
				ObjectiveID: objective.ID,
				Message:     message,
			})
			continue
		}

		remaining := *objective.BudgetRemainingPercent
		if remaining <= policy.Pause.RemainingBudgetAtOrBelowPercent {
			hasPause = true
			threshold := policy.Pause.RemainingBudgetAtOrBelowPercent
			decision.Reasons = append(decision.Reasons, Reason{
				Code:        "error_budget_exhausted",
				ObjectiveID: objective.ID,
				Observed:    floatPointer(remaining),
				Threshold:   floatPointer(threshold),
				Message:     fmt.Sprintf("remaining error budget %.6g%% is at or below the %.6g%% pause threshold", remaining, threshold),
			})
			continue
		}
		if remaining < policy.Caution.RemainingBudgetBelowPercent {
			hasCaution = true
			threshold := policy.Caution.RemainingBudgetBelowPercent
			decision.Reasons = append(decision.Reasons, Reason{
				Code:        "low_error_budget",
				ObjectiveID: objective.ID,
				Observed:    floatPointer(remaining),
				Threshold:   floatPointer(threshold),
				Message:     fmt.Sprintf("remaining error budget %.6g%% is below the %.6g%% caution threshold", remaining, threshold),
			})
		}
	}

	switch {
	case hasPause:
		decision.Recommendation = Pause
	case hasUnknown:
		decision.Recommendation = Unknown
	case hasCaution:
		decision.Recommendation = Caution
	default:
		decision.Recommendation = Proceed
		decision.Reasons = append(decision.Reasons, Reason{
			Code:    "within_policy",
			Message: "all objectives have usable evidence and remain within delivery policy thresholds",
		})
	}
	return decision
}

func uniqueSorted(values []string) []string {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if value != "" {
			seen[value] = struct{}{}
		}
	}
	result := make([]string, 0, len(seen))
	for value := range seen {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func floatPointer(value float64) *float64 {
	return &value
}
