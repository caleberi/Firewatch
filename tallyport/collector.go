package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Constants defining supported Prometheus metric types.
const (
	// _SUMMARY_ represents the Prometheus summary metric type.
	_SUMMARY_ = "summary"
	// _GAUGE_ represents the Prometheus gauge metric type.
	_GAUGE_ = "gauge"
	// _HISTOGRAM_ represents the Prometheus histogram metric type.
	_HISTOGRAM_ = "histogram"
	// _COUNTER_ represents the Prometheus counter metric type.
	_COUNTER_ = "counter"
)

// supportPrometheusType is a map indicating valid Prometheus metric types.
// Keys are metric type names (e.g., "counter", "gauge"), and values are true for supported types.
var supportPrometheusType = map[string]bool{
	_COUNTER_:   true,
	_HISTOGRAM_: true,
	_GAUGE_:     true,
	_SUMMARY_:   true,
}

// Metric represents a key for storing Prometheus metrics in a cache.
// The hash field uniquely identifies a metric by its name.
type Metric struct {
	hash string
}

// CacheMap is a thread-safe map for storing Prometheus metric vectors.
// It uses a generic type T to support different metric types (CounterVec, GaugeVec, etc.).
type CacheMap[T any] struct {
	sync.Mutex
	cache map[Metric]*T
}

// CollectorRegistry manages caches for different Prometheus metric types.
// It stores CounterVec, HistogramVec, GaugeVec, and SummaryVec instances in thread-safe maps.
type CollectorRegistry struct {
	counters   CacheMap[prometheus.CounterVec]
	histograms CacheMap[prometheus.HistogramVec]
	gauges     CacheMap[prometheus.GaugeVec]
	summary    CacheMap[prometheus.SummaryVec]
}

func NewCollectorRegistry() *CollectorRegistry {
	return &CollectorRegistry{
		counters:   CacheMap[prometheus.CounterVec]{cache: make(map[Metric]*prometheus.CounterVec)},
		histograms: CacheMap[prometheus.HistogramVec]{cache: make(map[Metric]*prometheus.HistogramVec)},
		gauges:     CacheMap[prometheus.GaugeVec]{cache: make(map[Metric]*prometheus.GaugeVec)},
		summary:    CacheMap[prometheus.SummaryVec]{cache: make(map[Metric]*prometheus.SummaryVec)},
	}
}

// BucketValue represents a single bucket configuration for a histogram metric.
// It includes a label and the upper bound value for the bucket.
type BucketValue struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

// MetricResponse defines the JSON response structure for metric operations.
// It contains a message describing the result of the operations
type MetricResponse struct {
	Message string `json:"message"`
}

// MetricRequest defines the JSON request structure for initializing or pushing metrics.
// It supports configuration for counter, gauge, histogram, and summary metric types.
type MetricRequest struct {
	Type        string   `json:"type"`                  // Type of the metric (counter, gauge, histogram, summary).
	Name        string   `json:"name"`                  // Unique name of the metric.
	Description string   `json:"description,omitempty"` // Description of the metric (optional for push).
	Labels      []string `json:"labels,omitempty"`      // Labels associated with the metric.
	Gauge       struct {
		Value float64 `json:"value,omitempty"` // Value for gauge metric updates.
	} `json:"gauge"` // Gauge-specific configuration.
	Histogram struct {
		Buckets       []float64 `json:"buckets,omitempty"`       // Bucket boundaries for histogram initialization.
		ObservedValue float64   `json:"observed_value,omitzero"` // Observed value for histogram updates.
	} `json:"histogram"` // Histogram-specific configuration.
	Summary struct {
		Objectives map[string]float64 `json:"objectives,omitempty"` // Quantile objectives for summary initialization.
		MaxAge     int64              `json:"max_age,omitempty"`    // Maximum age for summary observations.
	} `json:"summary"` // Summary-specific configuration.
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
		w.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight OPTIONS requests
		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// SetupRouter configures and returns a chi router for handling Prometheus metric operations.
// It sets up middleware for request handling, metrics collection, and endpoints for initializing and pushing metrics.
// The router includes:
// - /metrics: Exposes Prometheus metrics for scraping.
// - /init: Initializes a new metric (counter, gauge, histogram, or summary).
// - /push: Updates an existing metric with new values or observations.
//
// Parameters:
//   - reg: Prometheus registry for registering metrics.
//   - mc: CollectorRegistry for managing metric caches.
//
// Returns:
//   - *chi.Mux: Configured chi router instance.
func SetupRouter(reg *prometheus.Registry, mc *CollectorRegistry) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.NoCache)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.RequestSize(1024))
	r.Use(corsMiddleware)
	// TODO - Add rate limiter
	// TODO - Metric removal time after no usage for some time
	// TODO - Handle race condition with metric creation
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			start := time.Now()
			method := req.Method
			endpoint := req.URL.Path
			ww := middleware.NewWrapResponseWriter(res, req.ProtoMajor)
			next.ServeHTTP(ww, req)
			status := fmt.Sprintf("%d", ww.Status())

			{
				mc.counters.Lock()
				cache := mc.counters.cache[Metric{hash: "__tallyport__"}]
				cache.WithLabelValues(method, endpoint, status).Inc()
				mc.counters.Unlock()
			}

			{
				mc.histograms.Lock()
				cache := mc.histograms.cache[Metric{hash: "__tallyport__latency__"}]
				cache.WithLabelValues(method, endpoint).Observe(time.Since(start).Seconds())
				mc.histograms.Unlock()

			}
		})
	})

	r.Handle("/api/v1/metrics", promhttp.HandlerFor(
		reg,
		promhttp.HandlerOpts{
			Timeout:          30 * time.Second,
			ProcessStartTime: time.Now(),
		}))
	r.Post("/push", func(res http.ResponseWriter, req *http.Request) {
		var metricReq MetricRequest
		if !parseRequestBody(req, res, &metricReq) {
			return
		}

		if metricReq.Type == "" || metricReq.Name == "" {
			http.Error(res, "Missing required fields: type or name", http.StatusBadRequest)
			return
		}

		metricKey := Metric{hash: metricReq.Name}

		switch metricReq.Type {
		case _COUNTER_:
			mc.counters.Lock()
			defer mc.counters.Unlock()
			counter, exists := mc.counters.cache[metricKey]
			if !exists {
				http.Error(res, "Counter not found", http.StatusNotFound)
				return
			}
			counter.WithLabelValues(metricReq.Labels...).Inc()
			res.WriteHeader(http.StatusOK)
			json.NewEncoder(res).Encode(MetricResponse{
				Message: fmt.Sprintf("Counter %s updated", metricReq.Name),
			})
			return
		case _HISTOGRAM_:
			mc.histograms.Lock()
			defer mc.histograms.Unlock()
			histogram, exists := mc.histograms.cache[metricKey]
			if !exists {
				http.Error(res, "Histogram not found", http.StatusNotFound)
				return
			}

			if metricReq.Labels == nil {
				http.Error(res, "Labels for buckets are missing", http.StatusBadRequest)
				return
			}

			observedValue := metricReq.Histogram.ObservedValue
			histogram.WithLabelValues(metricReq.Labels...).Observe(observedValue)
			res.WriteHeader(http.StatusOK)
			json.NewEncoder(res).Encode(MetricResponse{
				Message: fmt.Sprintf("Histogram %s updated", metricReq.Name),
			})
		case _GAUGE_:
			mc.gauges.Lock()
			defer mc.gauges.Unlock()

			gauge, exists := mc.gauges.cache[metricKey]
			if !exists {
				http.Error(res, "Gauge not found", http.StatusNotFound)
				return
			}

			gauge.WithLabelValues(metricReq.Labels...).Set(metricReq.Gauge.Value)
			res.WriteHeader(http.StatusOK)
			json.NewEncoder(res).Encode(
				MetricResponse{
					Message: fmt.Sprintf("Gauge %s updated", metricReq.Name),
				})
		case _SUMMARY_:
			mc.summary.Lock()
			defer mc.summary.Unlock()
			summary, exists := mc.summary.cache[metricKey]
			if !exists {
				http.Error(res, "Summary not found", http.StatusNotFound)
				return
			}
			summary.WithLabelValues(metricReq.Labels...)
			res.WriteHeader(http.StatusOK)
			json.NewEncoder(res).Encode(
				MetricResponse{
					Message: fmt.Sprintf("Summary %s updated", metricReq.Name),
				})

		default:
			http.Error(res, "Invalid metric type", http.StatusBadRequest)
		}

	})

	r.Post("/init", func(res http.ResponseWriter, req *http.Request) {
		var metricReq MetricRequest
		if !parseRequestBody(req, res, &metricReq) {
			return
		}

		if metricReq.Type == "" || metricReq.Name == "" {
			http.Error(res, "Missing required fields: type, name, or description", http.StatusBadRequest)
			return
		}

		if !supportPrometheusType[metricReq.Type] {
			http.Error(res, "Invalid metric type", http.StatusBadRequest)
			return
		}

		metricKey := Metric{
			hash: metricReq.Name,
		}

		switch metricReq.Type {
		case _COUNTER_:
			mc.counters.Lock()
			defer mc.counters.Unlock()
			if _, exists := mc.counters.cache[metricKey]; exists {
				text := fmt.Sprintf("Metric already exists %v", metricKey)
				http.Error(res, text, http.StatusConflict)
				return
			}
			counter := prometheus.NewCounterVec(
				prometheus.CounterOpts{
					Name: metricReq.Name,
					Help: metricReq.Description,
				},
				metricReq.Labels,
			)
			mc.counters.cache[metricKey] = counter
			reg.MustRegister(counter)

		case _HISTOGRAM_:
			mc.histograms.Lock()
			defer mc.histograms.Unlock()
			if _, exists := mc.histograms.cache[metricKey]; exists {
				http.Error(res, "Metric already exists", http.StatusConflict)
				return
			}

			histogram := prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    metricReq.Name,
					Help:    metricReq.Description,
					Buckets: metricReq.Histogram.Buckets,
				},
				metricReq.Labels,
			)
			mc.histograms.cache[metricKey] = histogram
			reg.MustRegister(histogram)
		case _GAUGE_:
			mc.gauges.Lock()
			defer mc.gauges.Unlock()
			if _, exists := mc.gauges.cache[metricKey]; exists {
				http.Error(res, "Metric already exists", http.StatusConflict)
				return
			}
			gauge := prometheus.NewGaugeVec(
				prometheus.GaugeOpts{
					Name: metricReq.Name,
					Help: metricReq.Description,
				},
				metricReq.Labels,
			)
			mc.gauges.cache[metricKey] = gauge
			reg.MustRegister(gauge)

		case _SUMMARY_:
			mc.summary.Lock()
			defer mc.summary.Unlock()
			if _, exists := mc.summary.cache[metricKey]; exists {
				http.Error(res, "Metric already exists", http.StatusConflict)
				return
			}
			if metricReq.Summary.MaxAge <= 0 {
				http.Error(res, "Invalid MaxAge for summary", http.StatusBadRequest)
				return
			}
			objectives := map[float64]float64{}
			for kobj, vobj := range metricReq.Summary.Objectives {
				v, err := strconv.ParseFloat(kobj, 64)
				if err != nil {
					http.Error(res, "Invalid objectives data", http.StatusBadRequest)
					return
				}
				objectives[v] = vobj
			}
			summary := prometheus.NewSummaryVec(
				prometheus.SummaryOpts{
					Name:       metricReq.Name,
					Help:       metricReq.Description,
					Objectives: objectives,
					MaxAge:     time.Duration(metricReq.Summary.MaxAge),
				},
				metricReq.Labels,
			)
			mc.summary.cache[metricKey] = summary
			reg.MustRegister(summary)
		}

		res.WriteHeader(http.StatusCreated)
		fmt.Fprintf(res, "Metric %s created successfully", metricReq.Name)
	})
	return r
}
