// Package telemetry collects provider-specific measurements into the
// provider-neutral evidence interface consumed by the evaluation engine.
package telemetry

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/prometheus/prometheus/promql/parser"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/evaluation"
)

// PrometheusCollector executes contract query templates through the stable
// Prometheus HTTP API.
type PrometheusCollector struct {
	baseURL *url.URL
	client  *http.Client
}

// NewPrometheusCollector creates a collector with a bounded HTTP timeout.
func NewPrometheusCollector(rawURL string, timeout time.Duration) (*PrometheusCollector, error) {
	baseURL, err := url.Parse(rawURL)
	if err != nil {
		return nil, fmt.Errorf("parse Prometheus URL: %w", err)
	}
	if baseURL.Scheme != "http" && baseURL.Scheme != "https" {
		return nil, fmt.Errorf("Prometheus URL must use http or https")
	}
	if baseURL.Host == "" {
		return nil, fmt.Errorf("Prometheus URL must include a host")
	}
	return &PrometheusCollector{baseURL: baseURL, client: &http.Client{Timeout: timeout}}, nil
}

// Collect executes good and total event queries for every objective.
func (c *PrometheusCollector) Collect(ctx context.Context, reliability *contract.ServiceReliability, at time.Time) (*evaluation.Evidence, error) {
	evidence := &evaluation.Evidence{
		APIVersion:     "reliability.vaipex.io/v1alpha1",
		Kind:           "ReliabilityEvidence",
		Metadata:       evaluation.EvidenceMetadata{ServiceName: reliability.Metadata.Name},
		EvaluationTime: at.UTC(),
		Objectives:     make([]evaluation.ObjectiveEvidence, 0, len(reliability.Spec.Objectives)),
	}
	for _, objective := range reliability.Spec.Objectives {
		window, err := parseContractDuration(objective.Window)
		if err != nil {
			return nil, fmt.Errorf("objective %s window: %w", objective.ID, err)
		}
		goodQuery, err := renderQuery(objective.Indicator.GoodEventsQuery, objective.Window)
		if err != nil {
			return nil, fmt.Errorf("objective %s good-events query: %w", objective.ID, err)
		}
		totalQuery, err := renderQuery(objective.Indicator.TotalEventsQuery, objective.Window)
		if err != nil {
			return nil, fmt.Errorf("objective %s total-events query: %w", objective.ID, err)
		}
		good, goodObservedAt, err := c.query(ctx, goodQuery, at)
		if err != nil {
			return nil, fmt.Errorf("objective %s good-events query: %w", objective.ID, err)
		}
		total, totalObservedAt, err := c.query(ctx, totalQuery, at)
		if err != nil {
			return nil, fmt.Errorf("objective %s total-events query: %w", objective.ID, err)
		}
		observedAt := goodObservedAt
		if totalObservedAt.Before(observedAt) {
			observedAt = totalObservedAt
		}
		evidence.Objectives = append(evidence.Objectives, evaluation.ObjectiveEvidence{
			ObjectiveID: objective.ID,
			WindowStart: at.UTC().Add(-window),
			WindowEnd:   at.UTC(),
			ObservedAt:  observedAt.UTC(),
			GoodEvents:  good,
			TotalEvents: total,
		})
	}
	return evidence, nil
}

func (c *PrometheusCollector) query(ctx context.Context, expression string, at time.Time) (float64, time.Time, error) {
	endpoint := *c.baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + "/api/v1/query"
	parameters := endpoint.Query()
	parameters.Set("query", expression)
	parameters.Set("time", at.UTC().Format(time.RFC3339Nano))
	endpoint.RawQuery = parameters.Encode()

	request, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return 0, time.Time{}, err
	}
	response, err := c.client.Do(request)
	if err != nil {
		return 0, time.Time{}, err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return 0, time.Time{}, fmt.Errorf("Prometheus returned HTTP %d", response.StatusCode)
	}

	var payload queryResponse
	if err := json.NewDecoder(response.Body).Decode(&payload); err != nil {
		return 0, time.Time{}, fmt.Errorf("decode Prometheus response: %w", err)
	}
	if payload.Status != "success" {
		return 0, time.Time{}, fmt.Errorf("Prometheus query failed: %s: %s", payload.ErrorType, payload.Error)
	}
	if payload.Data.ResultType != "vector" {
		return 0, time.Time{}, fmt.Errorf("expected vector result, got %q", payload.Data.ResultType)
	}
	if len(payload.Data.Result) == 0 {
		return 0, at, nil
	}
	if len(payload.Data.Result) != 1 {
		return 0, time.Time{}, fmt.Errorf("expected one result series, got %d; aggregate the query to one value", len(payload.Data.Result))
	}
	value := payload.Data.Result[0].Value
	if len(value) != 2 {
		return 0, time.Time{}, fmt.Errorf("unexpected Prometheus sample shape")
	}
	timestamp, ok := value[0].(float64)
	if !ok {
		return 0, time.Time{}, fmt.Errorf("unexpected Prometheus timestamp type")
	}
	rawValue, ok := value[1].(string)
	if !ok {
		return 0, time.Time{}, fmt.Errorf("unexpected Prometheus value type")
	}
	numericValue, err := strconv.ParseFloat(rawValue, 64)
	if err != nil {
		return 0, time.Time{}, fmt.Errorf("parse Prometheus value %q: %w", rawValue, err)
	}
	return numericValue, time.Unix(0, int64(timestamp*float64(time.Second))).UTC(), nil
}

func renderQuery(template, window string) (string, error) {
	if !strings.Contains(template, "$window") {
		return "", fmt.Errorf("missing $window placeholder")
	}
	expression := strings.ReplaceAll(strings.TrimSpace(template), "$window", window)
	if _, err := parser.NewParser(parser.Options{}).ParseExpr(expression); err != nil {
		return "", fmt.Errorf("rendered PromQL is invalid: %w", err)
	}
	return expression, nil
}

func parseContractDuration(value string) (time.Duration, error) {
	if strings.HasSuffix(value, "d") {
		days, err := strconv.ParseInt(strings.TrimSuffix(value, "d"), 10, 64)
		if err != nil {
			return 0, err
		}
		return time.Duration(days) * 24 * time.Hour, nil
	}
	return time.ParseDuration(value)
}

type queryResponse struct {
	Status    string `json:"status"`
	ErrorType string `json:"errorType"`
	Error     string `json:"error"`
	Data      struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Value []any `json:"value"`
		} `json:"result"`
	} `json:"data"`
}
