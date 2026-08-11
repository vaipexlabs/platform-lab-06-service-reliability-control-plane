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
