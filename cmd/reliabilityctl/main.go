package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/decision"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/evaluation"
	prometheusrules "github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/rules"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/telemetry"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		printUsage(stderr)
		return 2
	}
	switch args[0] {
	case "validate":
		return runValidate(args[1:], stdout, stderr)
	case "evaluate":
		return runEvaluate(args[1:], stdout, stderr)
	case "collect":
		return runCollect(args[1:], stdout, stderr)
	case "render-rules":
		return runRenderRules(args[1:], stdout, stderr)
	case "decide":
		return runDecide(args[1:], stdout, stderr)
	default:
		printUsage(stderr)
		return 2
	}
}

func runValidate(args []string, stdout, stderr io.Writer) int {
	if len(args) != 1 {
		printUsage(stderr)
		return 2
	}

	validator, err := contract.NewValidator()
	if err != nil {
		fmt.Fprintf(stderr, "initialize validator: %v\n", err)
		return 2
	}

	document, err := validator.ValidateFile(args[0])
	if err != nil {
		fmt.Fprintf(stderr, "invalid: %s\n%v\n", args[0], err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"valid: %s (service=%s objectives=%d)\n",
		args[0],
		document.Metadata.Name,
		len(document.Spec.Objectives),
	)
	return 0
}

func runEvaluate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("evaluate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	contractPath := flags.String("contract", "", "path to a ServiceReliability YAML or JSON document")
	evidencePath := flags.String("evidence", "", "path to a ReliabilityEvidence YAML or JSON document")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *contractPath == "" || *evidencePath == "" || flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}

	validator, err := contract.NewValidator()
	if err != nil {
		fmt.Fprintf(stderr, "initialize contract validator: %v\n", err)
		return 2
	}
	reliability, err := validator.ValidateFile(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid contract: %s\n%v\n", *contractPath, err)
		return 1
	}

	engine, err := evaluation.NewEngine()
	if err != nil {
		fmt.Fprintf(stderr, "initialize evaluation engine: %v\n", err)
		return 2
	}
	result, err := engine.EvaluateFile(reliability, *evidencePath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid evidence: %s\n%v\n", *evidencePath, err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(result); err != nil {
		fmt.Fprintf(stderr, "encode evaluation: %v\n", err)
		return 2
	}
	return 0
}

func runCollect(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("collect", flag.ContinueOnError)
	flags.SetOutput(stderr)
	contractPath := flags.String("contract", "", "path to a ServiceReliability YAML or JSON document")
	prometheusURL := flags.String("prometheus-url", "http://localhost:9090", "Prometheus base URL")
	outputPath := flags.String("output", "-", "evidence output path, or - for stdout")
	atValue := flags.String("at", "", "optional RFC3339 evaluation time")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *contractPath == "" || flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}

	at := time.Now().UTC()
	if *atValue != "" {
		parsed, err := time.Parse(time.RFC3339, *atValue)
		if err != nil {
			fmt.Fprintf(stderr, "invalid --at value: %v\n", err)
			return 2
		}
		at = parsed.UTC()
	}
	validator, err := contract.NewValidator()
	if err != nil {
		fmt.Fprintf(stderr, "initialize contract validator: %v\n", err)
		return 2
	}
	reliability, err := validator.ValidateFile(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid contract: %s\n%v\n", *contractPath, err)
		return 1
	}
	collector, err := telemetry.NewPrometheusCollector(*prometheusURL, 10*time.Second)
	if err != nil {
		fmt.Fprintf(stderr, "initialize Prometheus collector: %v\n", err)
		return 2
	}
	evidence, err := collector.Collect(context.Background(), reliability, at)
	if err != nil {
		fmt.Fprintf(stderr, "collect evidence: %v\n", err)
		return 1
	}

	if *outputPath == "-" {
		if err := encodeJSON(stdout, evidence); err != nil {
			fmt.Fprintf(stderr, "encode evidence: %v\n", err)
			return 2
		}
		return 0
	}
	output, err := os.Create(*outputPath)
	if err != nil {
		fmt.Fprintf(stderr, "create evidence file: %v\n", err)
		return 2
	}
	if err := encodeJSON(output, evidence); err != nil {
		_ = output.Close()
		fmt.Fprintf(stderr, "encode evidence: %v\n", err)
		return 2
	}
	if err := output.Close(); err != nil {
		fmt.Fprintf(stderr, "close evidence file: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote evidence: %s\n", *outputPath)
	return 0
}

func runRenderRules(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("render-rules", flag.ContinueOnError)
	flags.SetOutput(stderr)
	contractPath := flags.String("contract", "", "path to a ServiceReliability YAML or JSON document")
	outputPath := flags.String("output", "-", "Prometheus rules output path, or - for stdout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *contractPath == "" || flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}
	validator, err := contract.NewValidator()
	if err != nil {
		fmt.Fprintf(stderr, "initialize contract validator: %v\n", err)
		return 2
	}
	reliability, err := validator.ValidateFile(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid contract: %s\n%v\n", *contractPath, err)
		return 1
	}
	data, err := prometheusrules.Generate(reliability)
	if err != nil {
		fmt.Fprintf(stderr, "render Prometheus rules: %v\n", err)
		return 1
	}
	if *outputPath == "-" {
		_, _ = stdout.Write(data)
		return 0
	}
	if err := os.WriteFile(*outputPath, data, 0o644); err != nil {
		fmt.Fprintf(stderr, "write Prometheus rules: %v\n", err)
		return 2
	}
	fmt.Fprintf(stdout, "wrote Prometheus rules: %s\n", *outputPath)
	return 0
}

func runDecide(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("decide", flag.ContinueOnError)
	flags.SetOutput(stderr)
	contractPath := flags.String("contract", "", "path to a ServiceReliability YAML or JSON document")
	evidencePath := flags.String("evidence", "", "path to a ReliabilityEvidence YAML or JSON document")
	var activeSeverities stringValues
	flags.Var(&activeSeverities, "active-severity", "active burn-rate severity; repeat for multiple values")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if *contractPath == "" || *evidencePath == "" || flags.NArg() != 0 {
		printUsage(stderr)
		return 2
	}

	validator, err := contract.NewValidator()
	if err != nil {
		fmt.Fprintf(stderr, "initialize contract validator: %v\n", err)
		return 2
	}
	reliability, err := validator.ValidateFile(*contractPath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid contract: %s\n%v\n", *contractPath, err)
		return 1
	}
	engine, err := evaluation.NewEngine()
	if err != nil {
		fmt.Fprintf(stderr, "initialize evaluation engine: %v\n", err)
		return 2
	}
	evaluationResult, err := engine.EvaluateFile(reliability, *evidencePath)
	if err != nil {
		fmt.Fprintf(stderr, "invalid evidence: %s\n%v\n", *evidencePath, err)
		return 1
	}
	result := decision.Decide(reliability, evaluationResult, activeSeverities)
	if err := encodeJSON(stdout, result); err != nil {
		fmt.Fprintf(stderr, "encode decision: %v\n", err)
		return 2
	}
	return 0
}

type stringValues []string

func (values *stringValues) String() string {
	return fmt.Sprint([]string(*values))
}

func (values *stringValues) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func encodeJSON(writer io.Writer, value any) error {
	encoder := json.NewEncoder(writer)
	encoder.SetIndent("", "  ")
	encoder.SetEscapeHTML(false)
	return encoder.Encode(value)
}

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage:")
	fmt.Fprintln(writer, "  reliabilityctl validate <contract.yaml|contract.json>")
	fmt.Fprintln(writer, "  reliabilityctl evaluate --contract <path> --evidence <path>")
	fmt.Fprintln(writer, "  reliabilityctl collect --contract <path> [--prometheus-url <url>] [--output <path>]")
	fmt.Fprintln(writer, "  reliabilityctl render-rules --contract <path> [--output <path>]")
	fmt.Fprintln(writer, "  reliabilityctl decide --contract <path> --evidence <path> [--active-severity <severity>]")
}
