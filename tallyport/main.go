package main

import (
	"encoding/json"
	"flag"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"tallyport/engine"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"
	"github.com/rs/zerolog"
	"gopkg.in/yaml.v2"
)

type TallyPortConfig struct {
	ServerConfig struct {
		MaxHeaderBytes    int    `yaml:"max_header_bytes"`
		ReadHeaderTimeout int64  `yaml:"read_header_timeout"`
		WriteTimeout      int64  `yaml:"write_timeout"`
		ReadTimeout       int64  `yaml:"read_timeout"`
		IdleTimeout       int64  `yaml:"idle_timeout"`
		ServerName        string `yaml:"server_name"`
		Port              string `yaml:"port"`
		TlsPath           string `yaml:"tls_path"`
	} `yaml:"server_config"`
}

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
		ReadHeaderTimeout:            time.Duration(config.ServerConfig.ReadHeaderTimeout),
		WriteTimeout:                 time.Duration(config.ServerConfig.WriteTimeout),
		IdleTimeout:                  time.Duration(config.ServerConfig.IdleTimeout),
		ReadTimeout:                  time.Duration(config.ServerConfig.ReadTimeout),
	}

	collectionRegistry := NewCollectorRegistry()
	key := Metric{hash: "__tallyport__"}
	collectionRegistry.counters.cache[key] = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "__tallyport__",
			Subsystem: "pushgateway",
			Name:      "tally_port_requests_total",
			Help:      "Track number of metrics processed by tallyport",
		},
		[]string{"method", "endpoint", "status"},
	)
	latencyKey := Metric{hash: "__tallyport__latency__"}
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
		config.ServerConfig.TlsPath, SetupRouter(reg, collectionRegistry), opts).Serve()
}

func parseRequestBody(r *http.Request, w http.ResponseWriter, metricReq *MetricRequest) bool {
	data, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return false
	}

	if err := json.Unmarshal(data, metricReq); err != nil {
		http.Error(w, "Invalid JSON format", http.StatusBadRequest)
		return false
	}

	return true
}

func fatalLog(err error, logger zerolog.Logger) {
	if err != nil {
		logger.Err(err).Stack().Send()
		os.Exit(1)
	}
}
