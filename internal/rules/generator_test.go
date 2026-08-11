package rules

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"gopkg.in/yaml.v3"
)

func TestGenerateProducesParseableRulesForEveryObjective(t *testing.T) {
	t.Parallel()

	validator, err := contract.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	reliability, err := validator.ValidateFile(filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	data, err := Generate(reliability)
	if err != nil {
		t.Fatal(err)
	}

	var file File
	if err := yaml.Unmarshal(data, &file); err != nil {
		t.Fatalf("generated YAML is invalid: %v", err)
	}
	if len(file.Groups) != 2 {
		t.Fatalf("groups = %d, want 2", len(file.Groups))
	}
	for _, group := range file.Groups {
		if len(group.Rules) != 24 {
			t.Fatalf("group %s rules = %d, want 24", group.Name, len(group.Rules))
		}
		for _, rule := range group.Rules {
			if _, err := parser.NewParser(parser.Options{}).ParseExpr(rule.Expr); err != nil {
				t.Errorf("group %s expression %q is invalid: %v", group.Name, rule.Expr, err)
			}
		}
	}
	text := string(data)
	for _, expected := range []string{
		"vaipex:slo_error_budget_remaining_ratio",
		"VaipexRequestAvailabilityFastBurn",
		"VaipexRequestLatencySlowBurn",
		"docs/runbooks/fast-burn.md",
	} {
		if !strings.Contains(text, expected) {
			t.Errorf("generated rules do not contain %q", expected)
		}
	}
}
