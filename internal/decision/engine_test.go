package decision

import (
	"testing"
	"time"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/evaluation"
)

func TestDecisionRecommendations(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		objectives []evaluation.ObjectiveResult
		active     []string
		want       Recommendation
		code       string
	}{
		{
			name:       "proceed",
			objectives: []evaluation.ObjectiveResult{objective("availability", "met", 50)},
			want:       Proceed,
			code:       "within_policy",
		},
		{
			name:       "caution",
			objectives: []evaluation.ObjectiveResult{objective("availability", "met", 10)},
			want:       Caution,
			code:       "low_error_budget",
		},
		{
			name:       "pause on exhausted budget",
			objectives: []evaluation.ObjectiveResult{objective("availability", "breached", -100)},
			want:       Pause,
			code:       "error_budget_exhausted",
		},
		{
			name:       "pause on configured alert severity",
			objectives: []evaluation.ObjectiveResult{objective("availability", "met", 50)},
			active:     []string{"page"},
			want:       Pause,
			code:       "active_burn_rate_severity",
		},
		{
			name: "unknown evidence",
			objectives: []evaluation.ObjectiveResult{{
				ID: "availability", Status: "unknown", Reason: "evidence is stale",
			}},
			want: Unknown,
			code: "unknown_evidence",
		},
		{
			name: "pause takes precedence over unknown",
			objectives: []evaluation.ObjectiveResult{{
				ID: "availability", Status: "unknown", Reason: "evidence missing",
			}},
			active: []string{"page"},
			want:   Pause,
			code:   "active_burn_rate_severity",
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			result := Decide(testContract(), &evaluation.Result{
				ServiceName: "checkout-service",
				EvaluatedAt: time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC),
				Objectives:  test.objectives,
			}, test.active)
			if result.Recommendation != test.want {
				t.Fatalf("recommendation = %q, want %q", result.Recommendation, test.want)
			}
			if !hasReason(result.Reasons, test.code) {
				t.Fatalf("reasons = %+v, want code %q", result.Reasons, test.code)
			}
		})
	}
}

func testContract() *contract.ServiceReliability {
	return &contract.ServiceReliability{Spec: contract.Spec{DecisionPolicy: contract.DecisionPolicy{
		Caution: contract.CautionPolicy{RemainingBudgetBelowPercent: 25},
		Pause: contract.PausePolicy{
			RemainingBudgetAtOrBelowPercent: 0,
			ActiveBurnRateSeverities:        []string{"page"},
		},
	}}}
}

func objective(id, status string, remaining float64) evaluation.ObjectiveResult {
	return evaluation.ObjectiveResult{
		ID: id, Status: status, BudgetRemainingPercent: &remaining,
	}
}

func hasReason(reasons []Reason, code string) bool {
	for _, reason := range reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
