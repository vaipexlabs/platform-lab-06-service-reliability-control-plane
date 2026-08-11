package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/evaluation"
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

func printUsage(writer io.Writer) {
	fmt.Fprintln(writer, "usage:")
	fmt.Fprintln(writer, "  reliabilityctl validate <contract.yaml|contract.json>")
	fmt.Fprintln(writer, "  reliabilityctl evaluate --contract <path> --evidence <path>")
}
