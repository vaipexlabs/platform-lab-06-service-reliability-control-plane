package evaluation

import (
	"fmt"
	"math"
	"path/filepath"
	"strings"
	"testing"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
)

func TestHealthyEvidence(t *testing.T) {
	t.Parallel()

	result := evaluateFixture(t, filepath.Join("..", "..", "contracts", "examples", "checkout-service-evidence.yaml"))
	if result.Summary != (Summary{Met: 2}) {
		t.Fatalf("summary = %+v, want 2 met", result.Summary)
	}
	assertFloat(t, result.Objectives[0].AttainmentPercent, 99.95)
	assertFloat(t, result.Objectives[0].AllowedBadFraction, 0.001)
	assertFloat(t, result.Objectives[0].ObservedBadFraction, 0.0005)
	assertFloat(t, result.Objectives[0].BurnRate, 0.5)
	assertFloat(t, result.Objectives[0].BudgetConsumedPercent, 50)
	assertFloat(t, result.Objectives[0].BudgetRemainingPercent, 50)
}

func TestDegradedEvidenceStillMeetsObjectivesWithLittleBudgetRemaining(t *testing.T) {
	t.Parallel()

	result := evaluateFixture(t, filepath.Join("..", "..", "testdata", "evidence", "degraded.yaml"))
	if result.Summary != (Summary{Met: 2}) {
		t.Fatalf("summary = %+v, want 2 met", result.Summary)
	}
	assertFloat(t, result.Objectives[0].BudgetRemainingPercent, 10)
	assertFloat(t, result.Objectives[1].BudgetRemainingPercent, 5)
}

func TestExhaustedEvidenceBreachesObjectives(t *testing.T) {
	t.Parallel()

	result := evaluateFixture(t, filepath.Join("..", "..", "testdata", "evidence", "exhausted.yaml"))
	if result.Summary != (Summary{Breached: 2}) {
		t.Fatalf("summary = %+v, want 2 breached", result.Summary)
	}
	assertFloat(t, result.Objectives[0].BurnRate, 2)
	assertFloat(t, result.Objectives[0].BudgetRemainingPercent, -100)
	assertFloat(t, result.Objectives[1].BurnRate, 2)
}

func TestMissingAndStaleEvidenceIsUnknown(t *testing.T) {
	t.Parallel()

	result := evaluateFixture(t, filepath.Join("..", "..", "testdata", "evidence", "missing-and-stale.yaml"))
	if result.Summary != (Summary{Unknown: 2}) {
		t.Fatalf("summary = %+v, want 2 unknown", result.Summary)
	}
	if result.Objectives[0].Reason != "evidence is stale" {
		t.Fatalf("availability reason = %q", result.Objectives[0].Reason)
	}
	if result.Objectives[1].Reason != "evidence missing" {
		t.Fatalf("latency reason = %q", result.Objectives[1].Reason)
	}
	if result.Objectives[0].BurnRate != nil {
		t.Fatal("unknown result must not expose a burn rate")
	}
}

func TestZeroAndInsufficientTrafficAreUnknown(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		goodEvents  float64
		totalEvents float64
		reason      string
	}{
		{name: "zero traffic", goodEvents: 0, totalEvents: 0, reason: "no traffic observed"},
		{name: "insufficient samples", goodEvents: 499, totalEvents: 500, reason: "insufficient samples"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			data := fmt.Sprintf(`
apiVersion: reliability.vaipex.io/v1alpha1
kind: ReliabilityEvidence
metadata:
  serviceName: checkout-service
evaluationTime: 2026-08-11T12:00:00Z
objectives:
  - objectiveId: request-availability
    windowStart: 2026-07-12T12:00:00Z
    windowEnd: 2026-08-11T12:00:00Z
    observedAt: 2026-08-11T11:58:00Z
    goodEvents: %v
    totalEvents: %v
`, test.goodEvents, test.totalEvents)

			result, err := newTestEngine(t).Evaluate(loadContract(t), []byte(data))
			if err != nil {
				t.Fatal(err)
			}
			if result.Objectives[0].Status != "unknown" || !strings.Contains(result.Objectives[0].Reason, test.reason) {
				t.Fatalf("objective = %+v, want unknown reason containing %q", result.Objectives[0], test.reason)
			}
		})
	}
}

func TestInvalidEvidenceRelationshipsAreRejected(t *testing.T) {
	t.Parallel()

	reliability := loadContract(t)
	engine := newTestEngine(t)
	data := `
apiVersion: reliability.vaipex.io/v1alpha1
kind: ReliabilityEvidence
metadata:
  serviceName: checkout-service
evaluationTime: 2026-08-11T12:00:00Z
objectives:
  - objectiveId: request-availability
    windowStart: 2026-07-12T12:00:00Z
    windowEnd: 2026-08-11T12:00:00Z
    observedAt: 2026-08-11T11:58:00Z
    goodEvents: 101
    totalEvents: 100
  - objectiveId: request-availability
    windowStart: 2026-07-12T12:00:00Z
    windowEnd: 2026-08-11T12:00:00Z
    observedAt: 2026-08-11T11:58:00Z
    goodEvents: 100
    totalEvents: 100
`
	_, err := engine.Evaluate(reliability, []byte(data))
	if err == nil {
		t.Fatal("invalid evidence accepted")
	}
	for _, expected := range []string{"must not exceed totalEvents", "duplicate objective"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error does not contain %q: %v", expected, err)
		}
	}
}

func evaluateFixture(t *testing.T, path string) *Result {
	t.Helper()
	result, err := newTestEngine(t).EvaluateFile(loadContract(t), path)
	if err != nil {
		t.Fatalf("EvaluateFile() error = %v", err)
	}
	return result
}

func loadContract(t *testing.T) *contract.ServiceReliability {
	t.Helper()
	validator, err := contract.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	document, err := validator.ValidateFile(filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return document
}

func newTestEngine(t *testing.T) *Engine {
	t.Helper()
	engine, err := NewEngine()
	if err != nil {
		t.Fatal(err)
	}
	return engine
}

func assertFloat(t *testing.T, actual *float64, expected float64) {
	t.Helper()
	if actual == nil {
		t.Fatalf("actual = nil, want %v", expected)
	}
	if math.Abs(*actual-expected) > 0.000001 {
		t.Fatalf("actual = %v, want %v", *actual, expected)
	}
}
