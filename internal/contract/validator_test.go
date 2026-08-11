package contract

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestValidCheckoutContract(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	document, err := validator.ValidateFile(filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml"))
	if err != nil {
		t.Fatalf("valid contract rejected: %v", err)
	}
	if document.Metadata.Name != "checkout-service" {
		t.Fatalf("metadata.name = %q, want checkout-service", document.Metadata.Name)
	}
	if len(document.Spec.Objectives) != 2 {
		t.Fatalf("len(objectives) = %d, want 2", len(document.Spec.Objectives))
	}
}

func TestJSONContractIsSupported(t *testing.T) {
	t.Parallel()

	yamlData, err := os.ReadFile(filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	var document any
	if err := yaml.Unmarshal(yamlData, &document); err != nil {
		t.Fatal(err)
	}
	jsonData, err := json.Marshal(document)
	if err != nil {
		t.Fatal(err)
	}

	if _, err := newTestValidator(t).Validate(jsonData); err != nil {
		t.Fatalf("valid JSON contract rejected: %v", err)
	}
}

func TestStructuralErrorsIncludeFieldPath(t *testing.T) {
	t.Parallel()

	_, err := newTestValidator(t).ValidateFile(filepath.Join("..", "..", "testdata", "contracts", "invalid-structure.yaml"))
	if err == nil {
		t.Fatal("invalid contract accepted")
	}
	for _, expected := range []string{"$.spec.service", "runbook", "$.spec.objectives[0].target"} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error %q does not contain %q", err, expected)
		}
	}
}

func TestSemanticErrorsAreCollected(t *testing.T) {
	t.Parallel()

	_, err := newTestValidator(t).ValidateFile(filepath.Join("..", "..", "testdata", "contracts", "invalid-semantics.yaml"))
	if err == nil {
		t.Fatal("invalid contract accepted")
	}
	for _, expected := range []string{
		"duplicates objective",
		"duplicates alert",
		"invalid PromQL",
		"must be longer than shortWindow",
		"unsupported provider/queryLanguage pair",
		"caution remaining-budget threshold",
	} {
		if !strings.Contains(err.Error(), expected) {
			t.Errorf("error does not contain %q:\n%s", expected, err)
		}
	}
}

func TestMalformedAndMultipleDocumentsAreRejected(t *testing.T) {
	t.Parallel()

	validator := newTestValidator(t)
	for name, data := range map[string]string{
		"malformed": "spec: [",
		"multiple":  "name: first\n---\nname: second\n",
		"empty":     "\n",
	} {
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			if _, err := validator.Validate([]byte(data)); err == nil {
				t.Fatal("invalid input accepted")
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	t.Parallel()

	duration, err := parseDuration("30d")
	if err != nil {
		t.Fatal(err)
	}
	if got, want := duration.Hours(), float64(30*24); got != want {
		t.Fatalf("30d hours = %v, want %v", got, want)
	}
}

func newTestValidator(t *testing.T) *Validator {
	t.Helper()
	validator, err := NewValidator()
	if err != nil {
		t.Fatalf("NewValidator() error = %v", err)
	}
	return validator
}
