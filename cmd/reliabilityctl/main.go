package main

import (
	"fmt"
	"io"
	"os"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	if len(args) != 2 || args[0] != "validate" {
		fmt.Fprintln(stderr, "usage: reliabilityctl validate <contract.yaml|contract.json>")
		return 2
	}

	validator, err := contract.NewValidator()
	if err != nil {
		fmt.Fprintf(stderr, "initialize validator: %v\n", err)
		return 2
	}

	document, err := validator.ValidateFile(args[1])
	if err != nil {
		fmt.Fprintf(stderr, "invalid: %s\n%v\n", args[1], err)
		return 1
	}

	fmt.Fprintf(
		stdout,
		"valid: %s (service=%s objectives=%d)\n",
		args[1],
		document.Metadata.Name,
		len(document.Spec.Objectives),
	)
	return 0
}
