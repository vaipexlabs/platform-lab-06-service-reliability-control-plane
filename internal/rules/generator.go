// Package rules renders Prometheus recording and alert rules from a validated
// service reliability contract.
package rules

import (
	"fmt"
	"strconv"
	"strings"
	"unicode"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"gopkg.in/yaml.v3"
)

const runbookBaseURL = "https://github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/blob/main/docs/runbooks/"

// File is a Prometheus rule file.
type File struct {
	Groups []Group `yaml:"groups"`
}

// Group contains the rules for one service objective.
type Group struct {
	Name     string `yaml:"name"`
	Interval string `yaml:"interval"`
	Rules    []Rule `yaml:"rules"`
}

// Rule represents either a recording rule or an alerting rule.
type Rule struct {
	Record      string            `yaml:"record,omitempty"`
	Alert       string            `yaml:"alert,omitempty"`
	Expr        string            `yaml:"expr"`
	For         string            `yaml:"for,omitempty"`
	Labels      map[string]string `yaml:"labels,omitempty"`
	Annotations map[string]string `yaml:"annotations,omitempty"`
}

// Generate renders deterministic Prometheus YAML from a validated contract.
func Generate(reliability *contract.ServiceReliability) ([]byte, error) {
	file := File{Groups: make([]Group, 0, len(reliability.Spec.Objectives))}
	for _, objective := range reliability.Spec.Objectives {
		group, err := objectiveGroup(reliability, objective)
		if err != nil {
			return nil, err
		}
		file.Groups = append(file.Groups, group)
	}
	data, err := yaml.Marshal(file)
	if err != nil {
		return nil, fmt.Errorf("marshal Prometheus rules: %w", err)
	}
	return append([]byte("# Generated from the ServiceReliability contract. Do not edit by hand.\n"), data...), nil
}

func objectiveGroup(reliability *contract.ServiceReliability, objective contract.Objective) (Group, error) {
	service := reliability.Metadata.Name
	group := Group{
		Name:     fmt.Sprintf("vaipex.%s.%s", service, objective.ID),
		Interval: reliability.Spec.Evaluation.Interval,
	}

	windows := []string{objective.Window}
	seen := map[string]bool{objective.Window: true}
	for _, alert := range objective.BurnRateAlerts {
		for _, window := range []string{alert.ShortWindow, alert.LongWindow} {
			if !seen[window] {
				seen[window] = true
				windows = append(windows, window)
			}
		}
	}

	for _, window := range windows {
		good, err := render(objective.Indicator.GoodEventsQuery, window)
		if err != nil {
			return Group{}, fmt.Errorf("objective %s good-events query for %s: %w", objective.ID, window, err)
		}
		total, err := render(objective.Indicator.TotalEventsQuery, window)
		if err != nil {
			return Group{}, fmt.Errorf("objective %s total-events query for %s: %w", objective.ID, window, err)
		}
		labels := objectiveLabels(reliability, objective, window)
		attainment := fmt.Sprintf("(%s) / (%s)", good, total)
		allowedBad := 1 - objective.Target/100
		burnRate := fmt.Sprintf("clamp_min(1 - (%s), 0) / %s", attainment, number(allowedBad))
		group.Rules = append(group.Rules,
			Rule{Record: "vaipex:slo_good_events", Expr: good, Labels: labels},
			Rule{Record: "vaipex:slo_total_events", Expr: total, Labels: labels},
			Rule{Record: "vaipex:slo_attainment_ratio", Expr: attainment, Labels: labels},
			Rule{Record: "vaipex:slo_burn_rate", Expr: burnRate, Labels: labels},
		)
	}

	baseLabels := objectiveLabels(reliability, objective, objective.Window)
	group.Rules = append(group.Rules,
		Rule{Record: "vaipex:slo_target_ratio", Expr: fmt.Sprintf("vector(%s)", number(objective.Target/100)), Labels: baseLabels},
		Rule{
			Record: "vaipex:slo_error_budget_remaining_ratio",
			Expr: fmt.Sprintf(
				"1 - vaipex:slo_burn_rate{service=%q,objective=%q,window=%q}",
				service,
				objective.ID,
				objective.Window,
			),
			Labels: baseLabels,
		},
	)

	for _, alert := range objective.BurnRateAlerts {
		alertLabels := map[string]string{
			"service":   service,
			"objective": objective.ID,
			"severity":  alert.Severity,
			"team":      reliability.Spec.Service.Owner.Team,
		}
		runbook := "slow-burn.md"
		if alert.Severity == "page" {
			runbook = "fast-burn.md"
		}
		group.Rules = append(group.Rules, Rule{
			Alert: alertRuleName(objective.ID, alert.Name),
			Expr: fmt.Sprintf(
				"(vaipex:slo_burn_rate{service=%q,objective=%q,window=%q} > %s) and on(service, objective) (vaipex:slo_burn_rate{service=%q,objective=%q,window=%q} > %s)",
				service, objective.ID, alert.ShortWindow, number(alert.Threshold),
				service, objective.ID, alert.LongWindow, number(alert.Threshold),
			),
			For:    "0m",
			Labels: alertLabels,
			Annotations: map[string]string{
				"summary":     fmt.Sprintf("%s is consuming the %s error budget too quickly", reliability.Spec.Service.DisplayName, objective.DisplayName),
				"description": fmt.Sprintf("Burn rate exceeded %s across %s and %s windows.", number(alert.Threshold), alert.ShortWindow, alert.LongWindow),
				"runbook_url": runbookBaseURL + runbook,
			},
		})
	}
	return group, nil
}

func objectiveLabels(reliability *contract.ServiceReliability, objective contract.Objective, window string) map[string]string {
	return map[string]string{
		"service":   reliability.Metadata.Name,
		"objective": objective.ID,
		"category":  objective.Indicator.Category,
		"team":      reliability.Spec.Service.Owner.Team,
		"window":    window,
	}
}

func render(template, window string) (string, error) {
	if !strings.Contains(template, "$window") {
		return "", fmt.Errorf("missing $window placeholder")
	}
	expression := strings.ReplaceAll(strings.TrimSpace(template), "$window", window)
	if _, err := parser.NewParser(parser.Options{}).ParseExpr(expression); err != nil {
		return "", fmt.Errorf("invalid rendered PromQL: %w", err)
	}
	return expression, nil
}

func alertRuleName(objectiveID, alertName string) string {
	return "Vaipex" + pascalCase(objectiveID) + pascalCase(alertName)
}

func pascalCase(value string) string {
	var result strings.Builder
	upperNext := true
	for _, character := range value {
		if !unicode.IsLetter(character) && !unicode.IsDigit(character) {
			upperNext = true
			continue
		}
		if upperNext {
			character = unicode.ToUpper(character)
			upperNext = false
		}
		result.WriteRune(character)
	}
	return result.String()
}

func number(value float64) string {
	return strconv.FormatFloat(value, 'f', -1, 64)
}
