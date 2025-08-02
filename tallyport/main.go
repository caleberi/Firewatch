package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"tallyport/engine"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/httprate"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

func main() {

	var (
		config     TallyPortConfig
		configFile string
	)
	flag.StringVar(&configFile, "config-file", "setting.yml", "configuration file for tallport server")
	flag.Parse()

	logger := zerolog.New(os.Stdout).With().Timestamp().Logger()
	path := filepath.Clean(configFile)

	file, err := os.Open(path)
	fatalLog(err, logger)
	defer file.Close()

	raw, err := io.ReadAll(file)
	fatalLog(err, logger)

	err = yaml.Unmarshal(raw, &config)
	fatalLog(err, logger)

	opts := engine.ServerOpts{
		EnableTls:                    false,
		DisableGeneralOptionsHandler: true,
		UseColorizedLogger:           true,
		MaxHeaderBytes:               config.ServerConfig.MaxHeaderBytes, // 1 MB
		ReadHeaderTimeout:            time.Duration(config.ServerConfig.ReadHeaderTimeout * int64(time.Millisecond)),
		WriteTimeout:                 time.Duration(config.ServerConfig.WriteTimeout * int64(time.Millisecond)),
		IdleTimeout:                  time.Duration(config.ServerConfig.IdleTimeout * int64(time.Millisecond)),
		ReadTimeout:                  time.Duration(config.ServerConfig.ReadTimeout * int64(time.Millisecond)),
	}

	collectionRegistry := NewCollectorRegistry()
	key := Metric{key: "__tallyport__"}
	collectionRegistry.counters.cache[key] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "__tallyport__",
			Subsystem: "pushgateway",
			Name:      "tally_port_requests_total",
			Help:      "Track number of metrics processed by tallyport",
		},
		[]string{"method", "endpoint", "status"},
	)
	latencyKey := Metric{key: "__tallyport__latency__"}
	collectionRegistry.histograms.cache[latencyKey] = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Namespace: "__tallyport__",
			Subsystem: "pushgateway",
			Name:      "request_latency_seconds",
			Help:      "Request latency in seconds",
			Buckets:   prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)
	reg := prometheus.NewRegistry()
	reg.MustRegister(
		collectionRegistry.counters.cache[key],
		collectionRegistry.histograms.cache[latencyKey],
		collectors.NewGoCollector(),
		collectors.NewProcessCollector(collectors.ProcessCollectorOpts{ReportErrors: true}),
	)

	engine.NewServer(
		config.ServerConfig.ServerName,
		config.ServerConfig.Port, logger,
		config.ServerConfig.TlsPath, setupRouter(config, reg, collectionRegistry), opts).Serve()
}

// setupRouter configures and returns a chi router for handling Prometheus metric operations.
// It sets up middleware for request handling, metrics collection, and endpoints for initializing and pushing metrics.
// The router includes:
// - /metrics: Exposes Prometheus metrics for scraping.
// - /init: Initializes a new metric (counter, gauge, histogram, or summary).
// - /push: Updates an existing metric with new values or observations.
//
// Parameters:
//   - cfg: Server configuration
//   - reg: Prometheus registry for registering metrics.
//   - mc: CollectorRegistry for managing metric caches.
//
// Returns:
//   - *chi.Mux: Configured chi router instance.
func setupRouter(cfg TallyPortConfig, reg *prometheus.Registry, mc *CollectorRegistry) *chi.Mux {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.NoCache)
	r.Use(middleware.Logger)
	r.Use(middleware.Recoverer)
	r.Use(corsMiddleware(cfg))
	r.Use(middleware.SupressNotFound(r))
	r.Use(middleware.RequestSize(cfg.RequestConfig.Size))
	r.Use(middleware.AllowContentType("application/json"))
	r.Use(middleware.Timeout(time.Duration(cfg.RequestConfig.Timeout)))
	r.Use(middleware.ThrottleWithOpts(middleware.ThrottleOpts{
		Limit:          cfg.ThrottleConfig.LimitSize,
		BacklogLimit:   cfg.ThrottleConfig.BacklogLimit,
		StatusCode:     cfg.ThrottleConfig.StatusCode,
		BacklogTimeout: cfg.ThrottleConfig.BacklogTimeout,
	}))
	r.Use(trackRequestMetric(mc))
	r.Use(middleware.Heartbeat(cfg.HeartBeatPath))

	r.Use(httprate.Limit(cfg.RateLimitSizePerMinute, time.Minute,
		httprate.WithLimitHandler(func(w http.ResponseWriter, r *http.Request) {
			response := MetricResponse{
				Status: http.StatusTooManyRequests,
				Reason: "Rate-limited. Hold on 😡. Don't bring me down.",
			}
			raw, err := response.ToJSON()
			if err != nil {
				http.Error(w, fmt.Sprintf("failed to marshal rate limit response: %v", err), http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
			w.Write(raw)
		}),
	))

	// TODO:  Work on metric removal with access time idea
	r.Handle(cfg.MetricExportPath,
		promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))
	r.Post("/push", PushStatRestMetric(mc, reg))
	r.Post("/init", RegisterRestMetric(mc, reg))

	return r
}

func corsMiddleware(cfg TallyPortConfig) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			for _, header := range cfg.CorsConfig.Headers {
				w.Header().Set(header.Key, header.Value)
			}

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func trackRequestMetric(mc *CollectorRegistry) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
			start := time.Now()
			method := req.Method
			endpoint := req.URL.Path
			ww := middleware.NewWrapResponseWriter(res, req.ProtoMajor)
			next.ServeHTTP(ww, req)
			status := fmt.Sprintf("%d", ww.Status())

			mc.counters.Lock()
			counter := mc.counters.cache[Metric{key: "__tallyport__"}]
			counter.WithLabelValues(method, endpoint, status).Inc()
			mc.counters.Unlock()

			mc.histograms.Lock()
			histogram := mc.histograms.cache[Metric{key: "__tallyport__latency__"}]
			histogram.WithLabelValues(method, endpoint).Observe(time.Since(start).Seconds())
			mc.histograms.Unlock()

		})
	}
}

func fatalLog(err error, logger zerolog.Logger) {
	if err != nil {
		logger.Err(err).Stack().Send()
		os.Exit(1)
	}
}
