package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	_SUMMARY_   = "summary"
	_GAUGE_     = "gauge"
	_HISTOGRAM_ = "histogram"
	_COUNTER_   = "counter"
)

var supportPrometheusType = map[string]bool{
	_COUNTER_:   true,
	_HISTOGRAM_: true,
	_GAUGE_:     true,
	_SUMMARY_:   true,
}

type Metric struct {
	hash        string
	description string
}

type CacheMap[T any] struct {
	sync.Mutex
	cache map[Metric]*T
}

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

type BucketValue struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type LabelValue struct {
	Label string  `json:"label"`
	Value float64 `json:"value"`
}

type MetricResponse struct {
	Message string `json:"message"`
}

type MetricRequest struct {
	Type        string   `json:"type"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Labels      []string `json:"labels,omitempty"`
	Gauge       struct {
		Label []string  `json:"label,omitempty"`
		Value []float64 `json:"value,omitempty"`
	} `json:"gauge"`
	Histogram struct {
		Buckets []BucketValue `json:"buckets,omitempty"`
	} `json:"histogram"`
	Summary struct {
		Objectives map[float64]float64 `json:"objectives,omitempty"`
		MaxAge     int64               `json:"max_age,omitempty"`
	} `json:"summary"`
}

func SetupRouter(reg *prometheus.Registry, mc *CollectorRegistry) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.NoCache)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.RequestSize(1024))

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

	r.Handle("/metrics", promhttp.Handler())
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

			for _, lv := range metricReq.Histogram.Buckets {
				histogram.WithLabelValues(lv.Label).Observe(lv.Value)
			}

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

			if len(metricReq.Gauge.Label) != len(metricReq.Gauge.Value) {
				http.Error(res, "Mismatched label and value counts", http.StatusBadRequest)
				return
			}
			for i, label := range metricReq.Gauge.Label {
				gauge.WithLabelValues(label).Set(metricReq.Gauge.Value[i])
			}
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

		if metricReq.Type == "" || metricReq.Name == "" || metricReq.Description == "" {
			http.Error(res, "Missing required fields: type, name, or description", http.StatusBadRequest)
			return
		}

		if !supportPrometheusType[metricReq.Type] {
			http.Error(res, "Invalid metric type", http.StatusBadRequest)
			return
		}

		metricKey := Metric{
			hash:        metricReq.Name,
			description: metricReq.Description,
		}

		switch metricReq.Type {
		case _COUNTER_:
			mc.counters.Lock()
			defer mc.counters.Unlock()
			if _, exists := mc.counters.cache[metricKey]; exists {
				http.Error(res, "Metric already exists", http.StatusConflict)
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
			buckets := []float64{}
			for _, v := range metricReq.Histogram.Buckets {
				buckets = append(buckets, v.Value)
			}
			histogram := prometheus.NewHistogramVec(
				prometheus.HistogramOpts{
					Name:    metricReq.Name,
					Help:    metricReq.Description,
					Buckets: buckets,
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
			summary := prometheus.NewSummaryVec(
				prometheus.SummaryOpts{
					Name:       metricReq.Name,
					Help:       metricReq.Description,
					Objectives: metricReq.Summary.Objectives,
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
