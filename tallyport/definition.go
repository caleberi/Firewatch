package main

import (
	"encoding/json"
	"fmt"
	"time"
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

// Metric represents a key for storing Prometheus metrics in a cache.
// The hash field uniquely identifies a metric by its name.
type Metric struct {
	key string
}

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
	CorsConfig struct {
		Headers []struct {
			Key   string `yaml:"key"`
			Value string `yaml:"value"`
		}
	} `yaml:"cors_config"`
	ThrottleConfig struct {
		LimitSize      int           `yaml:"limit"`
		BacklogLimit   int           `yaml:"backlog_limit"`
		BacklogTimeout time.Duration `yaml:"backlog_timeout"`
		StatusCode     int           `yaml:"status_code"`
	} `yaml:"throttle_config"`

	RequestConfig struct {
		Size    int64 `yaml:"request_size"`
		Timeout int64 `yaml:"request_timeout"`
	} `yaml:"request_config"`

	HeartBeatPath          string `yaml:"heart_beat_path"`
	MetricExportPath       string `yaml:"metric_export_path"`
	RateLimitSizePerMinute int    `yaml:"rate_limit_size_per_minute"`
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
	Status  int    `json:"status"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
}

func (mr MetricResponse) ToJSON() ([]byte, error) {
	data, err := json.Marshal(mr)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal MetricResponse to JSON: %v", err)
	}
	return data, nil
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
