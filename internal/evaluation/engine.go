// Package evaluation calculates SLO attainment and error-budget health from a
// validated service reliability contract and provider-neutral evidence.
package evaluation

import (
	"bytes"
	"encoding/json"
	"fmt"
	"math"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/contracts"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/document"
)

const evidenceSchemaName = "reliability-evidence.schema.json"

// Evidence is one deterministic, provider-neutral evaluation snapshot.
type Evidence struct {
	APIVersion     string              `json:"apiVersion"`
	Kind           string              `json:"kind"`
	Metadata       EvidenceMetadata    `json:"metadata"`
	EvaluationTime time.Time           `json:"evaluationTime"`
	Objectives     []ObjectiveEvidence `json:"objectives"`
}

// EvidenceMetadata connects evidence to one reliability contract.
type EvidenceMetadata struct {
	ServiceName string `json:"serviceName"`
}

// ObjectiveEvidence contains comparable good and total event values for one
// objective window.
type ObjectiveEvidence struct {
	ObjectiveID string    `json:"objectiveId"`
	WindowStart time.Time `json:"windowStart"`
	WindowEnd   time.Time `json:"windowEnd"`
	ObservedAt  time.Time `json:"observedAt"`
	GoodEvents  float64   `json:"goodEvents"`
	TotalEvents float64   `json:"totalEvents"`
}

// Result is the deterministic evaluation output emitted by reliabilityctl.
type Result struct {
	APIVersion  string            `json:"apiVersion"`
	Kind        string            `json:"kind"`
	ServiceName string            `json:"serviceName"`
	EvaluatedAt time.Time         `json:"evaluatedAt"`
	Objectives  []ObjectiveResult `json:"objectives"`
	Summary     Summary           `json:"summary"`
}

// ObjectiveResult explains one objective's status and all supporting math.
type ObjectiveResult struct {
	ID                     string   `json:"id"`
	Status                 string   `json:"status"`
	Reason                 string   `json:"reason,omitempty"`
	TargetPercent          float64  `json:"targetPercent"`
	GoodEvents             *float64 `json:"goodEvents,omitempty"`
	TotalEvents            *float64 `json:"totalEvents,omitempty"`
	AttainmentPercent      *float64 `json:"attainmentPercent,omitempty"`
	AllowedBadFraction     *float64 `json:"allowedBadFraction,omitempty"`
	ObservedBadFraction    *float64 `json:"observedBadFraction,omitempty"`
	BurnRate               *float64 `json:"burnRate,omitempty"`
	BudgetConsumedPercent  *float64 `json:"budgetConsumedPercent,omitempty"`
	BudgetRemainingPercent *float64 `json:"budgetRemainingPercent,omitempty"`
}

// Summary counts objective outcomes without hiding unknown evidence.
type Summary struct {
	Met      int `json:"met"`
	Breached int `json:"breached"`
	Unknown  int `json:"unknown"`
}

// Engine validates evidence and evaluates it against a reliability contract.
type Engine struct {
	evidenceSchema *jsonschema.Schema
}

// NewEngine compiles the embedded evidence schema.
func NewEngine() (*Engine, error) {
	data, err := contracts.Schemas.ReadFile(evidenceSchemaName)
	if err != nil {
		return nil, fmt.Errorf("read embedded evidence schema: %w", err)
	}
	documentValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode embedded evidence schema: %w", err)
	}
	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource(evidenceSchemaName, documentValue); err != nil {
		return nil, fmt.Errorf("register embedded evidence schema: %w", err)
	}
	schema, err := compiler.Compile(evidenceSchemaName)
	if err != nil {
		return nil, fmt.Errorf("compile embedded evidence schema: %w", err)
	}
	return &Engine{evidenceSchema: schema}, nil
}

// EvaluateFile reads evidence from disk and evaluates it.
func (e *Engine) EvaluateFile(reliability *contract.ServiceReliability, path string) (*Result, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read evidence: %w", err)
	}
	return e.Evaluate(reliability, data)
}

// Evaluate validates one evidence document and calculates every objective in
// the contract's declared order.
func (e *Engine) Evaluate(reliability *contract.ServiceReliability, data []byte) (*Result, error) {
	jsonData, err := document.Normalize(data)
	if err != nil {
		return nil, fmt.Errorf("decode evidence: %w", err)
	}
	documentValue, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("decode normalized evidence: %w", err)
	}
	if err := e.evidenceSchema.Validate(documentValue); err != nil {
		return nil, fmt.Errorf("evidence schema validation: %w", err)
	}

	var evidence Evidence
	if err := json.Unmarshal(jsonData, &evidence); err != nil {
		return nil, fmt.Errorf("decode typed evidence: %w", err)
	}
	if err := validateEvidence(reliability, &evidence); err != nil {
		return nil, err
	}

	freshness, err := parseDuration(reliability.Spec.Evaluation.DataFreshness)
	if err != nil {
		return nil, fmt.Errorf("parse data freshness: %w", err)
	}
	evidenceByObjective := make(map[string]ObjectiveEvidence, len(evidence.Objectives))
	for _, item := range evidence.Objectives {
		evidenceByObjective[item.ObjectiveID] = item
	}

	result := &Result{
		APIVersion:  "reliability.vaipex.io/v1alpha1",
		Kind:        "ReliabilityEvaluation",
		ServiceName: reliability.Metadata.Name,
		EvaluatedAt: evidence.EvaluationTime,
		Objectives:  make([]ObjectiveResult, 0, len(reliability.Spec.Objectives)),
	}
	for _, objective := range reliability.Spec.Objectives {
		item, found := evidenceByObjective[objective.ID]
		objectiveResult := evaluateObjective(objective, item, found, evidence.EvaluationTime, freshness)
		result.Objectives = append(result.Objectives, objectiveResult)
		switch objectiveResult.Status {
		case "met":
			result.Summary.Met++
		case "breached":
			result.Summary.Breached++
		default:
			result.Summary.Unknown++
		}
	}
	return result, nil
}

func validateEvidence(reliability *contract.ServiceReliability, evidence *Evidence) error {
	if evidence.Metadata.ServiceName != reliability.Metadata.Name {
		return fmt.Errorf("evidence service %q does not match contract service %q", evidence.Metadata.ServiceName, reliability.Metadata.Name)
	}
	known := make(map[string]struct{}, len(reliability.Spec.Objectives))
	for _, objective := range reliability.Spec.Objectives {
		known[objective.ID] = struct{}{}
	}
	seen := map[string]struct{}{}
	var problems []string
	for i, item := range evidence.Objectives {
		path := fmt.Sprintf("$.objectives[%d]", i)
		if _, exists := seen[item.ObjectiveID]; exists {
			problems = append(problems, fmt.Sprintf("%s.objectiveId: duplicate objective %q", path, item.ObjectiveID))
		}
		seen[item.ObjectiveID] = struct{}{}
		if _, exists := known[item.ObjectiveID]; !exists {
			problems = append(problems, fmt.Sprintf("%s.objectiveId: objective %q is not declared by the contract", path, item.ObjectiveID))
		}
		if item.GoodEvents > item.TotalEvents {
			problems = append(problems, fmt.Sprintf("%s.goodEvents: must not exceed totalEvents", path))
		}
		if item.WindowEnd.Before(item.WindowStart) || item.WindowEnd.Equal(item.WindowStart) {
			problems = append(problems, fmt.Sprintf("%s.windowEnd: must be after windowStart", path))
		}
	}
	if len(problems) > 0 {
		sort.Strings(problems)
		return fmt.Errorf("invalid evidence:\n%s", strings.Join(problems, "\n"))
	}
	return nil
}

func evaluateObjective(objective contract.Objective, evidence ObjectiveEvidence, found bool, evaluationTime time.Time, freshness time.Duration) ObjectiveResult {
	result := ObjectiveResult{ID: objective.ID, Status: "unknown", TargetPercent: objective.Target}
	if !found {
		result.Reason = "evidence missing"
		return result
	}
	result.GoodEvents = floatPointer(evidence.GoodEvents)
	result.TotalEvents = floatPointer(evidence.TotalEvents)

	if evidence.ObservedAt.After(evaluationTime) {
		result.Reason = "evidence timestamp is in the future"
		return result
	}
	if evaluationTime.Sub(evidence.ObservedAt) > freshness {
		result.Reason = "evidence is stale"
		return result
	}
	objectiveWindow, err := parseDuration(objective.Window)
	if err != nil || evidence.WindowEnd.Sub(evidence.WindowStart) != objectiveWindow {
		result.Reason = "evidence window does not match objective window"
		return result
	}
	if !evidence.WindowEnd.Equal(evaluationTime) {
		result.Reason = "evidence window is not aligned with evaluation time"
		return result
	}
	if evidence.TotalEvents == 0 {
		result.Reason = "no traffic observed"
		return result
	}
	if objective.MinimumTotalEvents != nil && evidence.TotalEvents < float64(*objective.MinimumTotalEvents) {
		result.Reason = fmt.Sprintf("insufficient samples: got %s, require at least %d", formatNumber(evidence.TotalEvents), *objective.MinimumTotalEvents)
		return result
	}

	attainment := evidence.GoodEvents / evidence.TotalEvents
	allowedBad := 1 - objective.Target/100
	observedBad := 1 - attainment
	burnRate := observedBad / allowedBad
	consumed := burnRate * 100
	remaining := (1 - burnRate) * 100

	result.AttainmentPercent = floatPointer(round(attainment * 100))
	result.AllowedBadFraction = floatPointer(round(allowedBad))
	result.ObservedBadFraction = floatPointer(round(observedBad))
	result.BurnRate = floatPointer(round(burnRate))
	result.BudgetConsumedPercent = floatPointer(round(consumed))
	result.BudgetRemainingPercent = floatPointer(round(remaining))
	result.Status = "met"
	if attainment < objective.Target/100 {
		result.Status = "breached"
	}
	return result
}

func parseDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseInt(strings.TrimSuffix(value, "d"), 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(value)
}

func round(value float64) float64 {
	return math.Round(value*1_000_000) / 1_000_000
}

func floatPointer(value float64) *float64 {
	return &value
}

func formatNumber(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
