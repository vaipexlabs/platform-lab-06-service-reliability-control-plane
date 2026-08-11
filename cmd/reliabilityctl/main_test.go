package main

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunValidContract(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	path := filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml")

	if code := run([]string{"validate", path}, &stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %s", code, stderr.String())
	}
	if !strings.Contains(stdout.String(), "service=checkout-service objectives=2") {
		t.Fatalf("stdout = %q", stdout.String())
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}
}

func TestRunEvaluate(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	contractPath := filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml")
	evidencePath := filepath.Join("..", "..", "contracts", "examples", "checkout-service-evidence.yaml")
	args := []string{"evaluate", "--contract", contractPath, "--evidence", evidencePath}

	if code := run(args, &stdout, &stderr); code != 0 {
		t.Fatalf("run() code = %d, want 0; stderr = %s", code, stderr.String())
	}
	for _, expected := range []string{`"kind": "ReliabilityEvaluation"`, `"met": 2`, `"unknown": 0`} {
		if !strings.Contains(stdout.String(), expected) {
			t.Errorf("stdout does not contain %q: %s", expected, stdout.String())
		}
	}
}

func TestRunDecide(t *testing.T) {
	t.Parallel()

	contractPath := filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml")
	tests := []struct {
		name     string
		evidence string
		extra    []string
		want     string
	}{
		{
			name:     "proceed",
			evidence: filepath.Join("..", "..", "contracts", "examples", "checkout-service-evidence.yaml"),
			want:     `"recommendation": "proceed"`,
		},
		{
			name:     "caution",
			evidence: filepath.Join("..", "..", "testdata", "evidence", "degraded.yaml"),
			want:     `"recommendation": "caution"`,
		},
		{
			name:     "pause",
			evidence: filepath.Join("..", "..", "testdata", "evidence", "exhausted.yaml"),
			want:     `"recommendation": "pause"`,
		},
		{
			name:     "active severity pauses a healthy evaluation",
			evidence: filepath.Join("..", "..", "contracts", "examples", "checkout-service-evidence.yaml"),
			extra:    []string{"--active-severity", "page"},
			want:     `"recommendation": "pause"`,
		},
	}

	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			args := []string{"decide", "--contract", contractPath, "--evidence", test.evidence}
			args = append(args, test.extra...)
			if code := run(args, &stdout, &stderr); code != 0 {
				t.Fatalf("run() code = %d, want 0; stderr = %s", code, stderr.String())
			}
			if !strings.Contains(stdout.String(), test.want) {
				t.Fatalf("stdout does not contain %q: %s", test.want, stdout.String())
			}
		})
	}
}

func TestRunInvalidContract(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	path := filepath.Join("..", "..", "testdata", "contracts", "invalid-semantics.yaml")

	if code := run([]string{"validate", path}, &stdout, &stderr); code != 1 {
		t.Fatalf("run() code = %d, want 1", code)
	}
	if !strings.Contains(stderr.String(), "invalid PromQL") {
		t.Fatalf("stderr = %q", stderr.String())
	}
	if stdout.Len() != 0 {
		t.Fatalf("stdout = %q, want empty", stdout.String())
	}
}

func TestRunUsage(t *testing.T) {
	t.Parallel()

	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 {
		t.Fatalf("run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("stderr = %q", stderr.String())
	}
}
