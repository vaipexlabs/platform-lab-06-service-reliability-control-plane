package telemetry

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/vaipexlabs/platform-lab-06-service-reliability-control-plane/internal/contract"
)

func TestCollectRendersObjectiveWindowAndBuildsEvidence(t *testing.T) {
	t.Parallel()

	at := time.Date(2026, 8, 11, 12, 0, 0, 0, time.UTC)
	var queries []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query().Get("query")
		queries = append(queries, query)
		value := "10000"
		if strings.Contains(query, `status!="error"`) {
			value = "9995"
		} else if strings.Contains(query, `_bucket`) {
			value = "9950"
		}
		fmt.Fprintf(w, `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1786459200,%q]}]}}`, value)
	}))
	defer server.Close()

	collector, err := NewPrometheusCollector(server.URL, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	evidence, err := collector.Collect(context.Background(), loadContract(t), at)
	if err != nil {
		t.Fatal(err)
	}
	if len(queries) != 4 {
		t.Fatalf("queries = %d, want 4", len(queries))
	}
	for _, query := range queries {
		if !strings.Contains(query, "[30d]") || strings.Contains(query, "$window") {
			t.Fatalf("query was not rendered with objective window: %q", query)
		}
	}
	if got := evidence.Objectives[0].GoodEvents; got != 9995 {
		t.Fatalf("availability good events = %v, want 9995", got)
	}
	if got := evidence.Objectives[1].GoodEvents; got != 9950 {
		t.Fatalf("latency good events = %v, want 9950", got)
	}
	if got := evidence.Objectives[0].WindowEnd.Sub(evidence.Objectives[0].WindowStart); got != 30*24*time.Hour {
		t.Fatalf("window = %v, want 30d", got)
	}
}

func TestPrometheusFailuresAreActionable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		response string
		status   int
		expect   string
	}{
		{name: "api error", status: http.StatusOK, response: `{"status":"error","errorType":"bad_data","error":"invalid query"}`, expect: "bad_data: invalid query"},
		{name: "multiple series", status: http.StatusOK, response: `{"status":"success","data":{"resultType":"vector","result":[{"value":[1,"1"]},{"value":[1,"2"]}]}}`, expect: "aggregate the query"},
		{name: "http error", status: http.StatusServiceUnavailable, response: `{}`, expect: "HTTP 503"},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(test.status)
				_, _ = w.Write([]byte(test.response))
			}))
			defer server.Close()
			collector, err := NewPrometheusCollector(server.URL, time.Second)
			if err != nil {
				t.Fatal(err)
			}
			_, _, err = collector.query(context.Background(), "sum(up)", time.Now())
			if err == nil || !strings.Contains(err.Error(), test.expect) {
				t.Fatalf("error = %v, want containing %q", err, test.expect)
			}
		})
	}
}

func TestCollectorURLValidation(t *testing.T) {
	t.Parallel()

	for _, rawURL := range []string{"localhost:9090", "file:///tmp/prometheus"} {
		if _, err := NewPrometheusCollector(rawURL, time.Second); err == nil {
			t.Fatalf("URL %q accepted", rawURL)
		}
	}
}

func loadContract(t *testing.T) *contract.ServiceReliability {
	t.Helper()
	validator, err := contract.NewValidator()
	if err != nil {
		t.Fatal(err)
	}
	reliability, err := validator.ValidateFile(filepath.Join("..", "..", "contracts", "examples", "checkout-service.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	return reliability
}
