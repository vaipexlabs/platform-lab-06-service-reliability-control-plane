// checkout-demo is a deliberately small instrumented service used to produce
// repeatable reliability evidence for the local reference journey.
package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const serviceName = "checkout-service"

var (
	requests = prometheus.NewCounterVec(prometheus.CounterOpts{
		Name: "vaipex_checkout_requests_total",
		Help: "Completed checkout requests by outcome.",
	}, []string{"service", "status"})
	duration = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "vaipex_checkout_request_duration_seconds",
		Help:    "Checkout request duration in seconds.",
		Buckets: []float64{0.05, 0.1, 0.25, 0.5, 1, 2},
	}, []string{"service"})
)

func main() {
	prometheus.MustRegister(requests, duration)
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/health/ready", func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	mux.HandleFunc("/checkout", checkout)
	mux.HandleFunc("/demo/traffic", generateTraffic)

	server := &http.Server{Addr: ":8080", Handler: mux, ReadHeaderTimeout: 5 * time.Second}
	log.Printf("checkout demo listening on %s", server.Addr)
	log.Fatal(server.ListenAndServe())
}

func checkout(w http.ResponseWriter, r *http.Request) {
	status := r.URL.Query().Get("status")
	if status != "error" {
		status = "success"
	}
	latencyMilliseconds := queryInteger(r, "latency_ms", 100, 0, 5000)
	record(status, time.Duration(latencyMilliseconds)*time.Millisecond)
	if status == "error" {
		http.Error(w, "checkout failed", http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "accepted"})
}

func generateTraffic(w http.ResponseWriter, r *http.Request) {
	total := queryInteger(r, "requests", 2000, 1, 100000)
	failures := queryInteger(r, "failures", minimum(1, total), 0, total)
	slow := queryInteger(r, "slow", minimum(10, total), 0, total)
	for i := 0; i < total; i++ {
		status := "success"
		if i < failures {
			status = "error"
		}
		latency := 100 * time.Millisecond
		if i < slow {
			latency = 800 * time.Millisecond
		}
		record(status, latency)
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(map[string]int{"requests": total, "failures": failures, "slow": slow})
}

func minimum(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func record(status string, elapsed time.Duration) {
	requests.WithLabelValues(serviceName, status).Inc()
	duration.WithLabelValues(serviceName).Observe(elapsed.Seconds())
}

func queryInteger(r *http.Request, name string, fallback, minimum, maximum int) int {
	raw := r.URL.Query().Get(name)
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil || value < minimum || value > maximum {
		return fallback
	}
	return value
}

func init() {
	requests.WithLabelValues(serviceName, "success")
	requests.WithLabelValues(serviceName, "error")
	duration.WithLabelValues(serviceName)
}
