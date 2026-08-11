package contract

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"
	"time"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/contracts"
	"gopkg.in/yaml.v3"
)

const schemaName = "service-reliability.schema.json"

// Problem is one field-level structural or semantic validation failure.
type Problem struct {
	Path    string
	Message string
}

// ValidationError contains every problem found in one contract.
type ValidationError struct {
	Problems []Problem
}

func (e *ValidationError) Error() string {
	lines := make([]string, 0, len(e.Problems))
	for _, problem := range e.Problems {
		lines = append(lines, fmt.Sprintf("%s: %s", problem.Path, problem.Message))
	}
	return strings.Join(lines, "\n")
}

// Validator applies the embedded JSON Schema and the v1alpha1 semantic rules.
type Validator struct {
	schema *jsonschema.Schema
}

// NewValidator compiles the embedded Draft 2020-12 schema and enables URI
// format checks.
func NewValidator() (*Validator, error) {
	data, err := contracts.Schemas.ReadFile(schemaName)
	if err != nil {
		return nil, fmt.Errorf("read embedded schema: %w", err)
	}

	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("decode embedded schema: %w", err)
	}

	compiler := jsonschema.NewCompiler()
	compiler.DefaultDraft(jsonschema.Draft2020)
	compiler.AssertFormat()
	if err := compiler.AddResource(schemaName, document); err != nil {
		return nil, fmt.Errorf("register embedded schema: %w", err)
	}

	schema, err := compiler.Compile(schemaName)
	if err != nil {
		return nil, fmt.Errorf("compile embedded schema: %w", err)
	}
	return &Validator{schema: schema}, nil
}

// ValidateFile reads and validates one YAML or JSON contract.
func (v *Validator) ValidateFile(path string) (*ServiceReliability, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read contract: %w", err)
	}
	return v.Validate(data)
}

// Validate parses one YAML or JSON document, applies the schema, then applies
// semantic checks that JSON Schema cannot express clearly.
func (v *Validator) Validate(data []byte) (*ServiceReliability, error) {
	jsonData, err := yamlToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("decode contract: %w", err)
	}

	document, err := jsonschema.UnmarshalJSON(bytes.NewReader(jsonData))
	if err != nil {
		return nil, fmt.Errorf("decode normalized contract: %w", err)
	}
	if err := v.schema.Validate(document); err != nil {
		return nil, structuralError(err)
	}

	var contract ServiceReliability
	if err := json.Unmarshal(jsonData, &contract); err != nil {
		return nil, fmt.Errorf("decode typed contract: %w", err)
	}

	problems := validateSemantics(&contract)
	if len(problems) > 0 {
		sort.SliceStable(problems, func(i, j int) bool {
			if problems[i].Path == problems[j].Path {
				return problems[i].Message < problems[j].Message
			}
			return problems[i].Path < problems[j].Path
		})
		return nil, &ValidationError{Problems: problems}
	}
	return &contract, nil
}

func yamlToJSON(data []byte) ([]byte, error) {
	decoder := yaml.NewDecoder(bytes.NewReader(data))
	var document any
	if err := decoder.Decode(&document); err != nil {
		return nil, err
	}
	if document == nil {
		return nil, errors.New("document is empty")
	}

	var trailing any
	if err := decoder.Decode(&trailing); err != io.EOF {
		if err == nil {
			return nil, errors.New("multiple YAML or JSON documents are not allowed")
		}
		return nil, err
	}
	return json.Marshal(document)
}

func structuralError(err error) error {
	var validationErr *jsonschema.ValidationError
	if !errors.As(err, &validationErr) {
		return fmt.Errorf("schema validation: %w", err)
	}

	var leaves []*jsonschema.ValidationError
	collectValidationLeaves(validationErr, &leaves)
	problems := make([]Problem, 0, len(leaves))
	for _, leaf := range leaves {
		path := tokensToPath(leaf.InstanceLocation)
		problems = append(problems, Problem{Path: path, Message: leaf.Error()})
	}
	if len(problems) == 0 {
		return fmt.Errorf("schema validation: %w", err)
	}
	return &ValidationError{Problems: problems}
}

func collectValidationLeaves(err *jsonschema.ValidationError, leaves *[]*jsonschema.ValidationError) {
	if len(err.Causes) == 0 {
		*leaves = append(*leaves, err)
		return
	}
	for _, cause := range err.Causes {
		collectValidationLeaves(cause, leaves)
	}
}

func tokensToPath(parts []string) string {
	if len(parts) == 0 {
		return "$"
	}
	var result strings.Builder
	result.WriteString("$")
	for _, part := range parts {
		if isDigits(part) {
			fmt.Fprintf(&result, "[%s]", part)
		} else {
			result.WriteString(".")
			result.WriteString(part)
		}
	}
	return result.String()
}

func isDigits(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func validateSemantics(contract *ServiceReliability) []Problem {
	var problems []Problem
	evaluationInterval, _ := parseDuration(contract.Spec.Evaluation.Interval)

	objectiveNames := map[string]int{}
	for i, objective := range contract.Spec.Objectives {
		base := fmt.Sprintf("$.spec.objectives[%d]", i)
		if previous, exists := objectiveNames[objective.ID]; exists {
			problems = append(problems, Problem{
				Path:    base + ".id",
				Message: fmt.Sprintf("duplicates objective %q declared at index %d", objective.ID, previous),
			})
		} else {
			objectiveNames[objective.ID] = i
		}

		objectiveWindow, _ := parseDuration(objective.Window)
		if evaluationInterval >= objectiveWindow {
			problems = append(problems, Problem{
				Path:    base + ".window",
				Message: "must be longer than spec.evaluation.interval",
			})
		}

		if objective.Indicator.Provider != "prometheus" || objective.Indicator.QueryLanguage != "promql" {
			problems = append(problems, Problem{
				Path: base + ".indicator",
				Message: fmt.Sprintf(
					"unsupported provider/queryLanguage pair %q/%q; supported pair is prometheus/promql",
					objective.Indicator.Provider,
					objective.Indicator.QueryLanguage,
				),
			})
		} else {
			problems = append(problems, validatePromQL(base+".indicator.goodEventsQuery", objective.Indicator.GoodEventsQuery)...)
			problems = append(problems, validatePromQL(base+".indicator.totalEventsQuery", objective.Indicator.TotalEventsQuery)...)
		}

		alertNames := map[string]int{}
		for j, alert := range objective.BurnRateAlerts {
			alertBase := fmt.Sprintf("%s.burnRateAlerts[%d]", base, j)
			if previous, exists := alertNames[alert.Name]; exists {
				problems = append(problems, Problem{
					Path:    alertBase + ".name",
					Message: fmt.Sprintf("duplicates alert %q declared at index %d", alert.Name, previous),
				})
			} else {
				alertNames[alert.Name] = j
			}

			shortWindow, _ := parseDuration(alert.ShortWindow)
			longWindow, _ := parseDuration(alert.LongWindow)
			if longWindow <= shortWindow {
				problems = append(problems, Problem{
					Path:    alertBase + ".longWindow",
					Message: "must be longer than shortWindow",
				})
			}
			if evaluationInterval >= shortWindow {
				problems = append(problems, Problem{
					Path:    alertBase + ".shortWindow",
					Message: "must be longer than spec.evaluation.interval",
				})
			}
			if longWindow > objectiveWindow {
				problems = append(problems, Problem{
					Path:    alertBase + ".longWindow",
					Message: "must not exceed the objective window",
				})
			}
		}
	}

	caution := contract.Spec.DecisionPolicy.Caution.RemainingBudgetBelowPercent
	pause := contract.Spec.DecisionPolicy.Pause.RemainingBudgetAtOrBelowPercent
	if pause > caution {
		problems = append(problems, Problem{
			Path:    "$.spec.decisionPolicy.pause.remainingBudgetAtOrBelowPercent",
			Message: "must be less than or equal to the caution remaining-budget threshold",
		})
	}
	return problems
}

func validatePromQL(path, query string) []Problem {
	expression, err := parser.NewParser(parser.Options{}).ParseExpr(strings.TrimSpace(query))
	if err != nil {
		return []Problem{{Path: path, Message: fmt.Sprintf("invalid PromQL: %v", err)}}
	}
	if expression == nil {
		return []Problem{{Path: path, Message: "invalid PromQL: expression is empty"}}
	}
	return nil
}

func parseDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days := strings.TrimSuffix(value, "d")
		parsed, err := time.ParseDuration(days + "h")
		if err != nil {
			return 0, err
		}
		return parsed * 24, nil
	}
	return time.ParseDuration(value)
}
