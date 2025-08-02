package main

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

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

func (mc *CollectorRegistry) update(metric MetricRequest) error {
	validator := NewValidator(metric)
	metricKey := Metric{key: metric.Name}

	if metric.Type == _COUNTER_ {
		mc.counters.Lock()
		defer mc.counters.Unlock()

		if counter, exists := mc.counters.cache[metricKey]; exists {
			if err := validator.ValidateField("Labels", IsEmpty).Errors(); err != nil {
				raw, err := err.ToJSON()
				if err != nil {
					return err
				}
				return errors.New(string(raw))
			}
			counter.WithLabelValues(metric.Labels...).Inc()
			return nil
		}
		return fmt.Errorf("counter not found: %v", metricKey)
	}

	if metric.Type == _HISTOGRAM_ {
		mc.histograms.Lock()
		defer mc.histograms.Unlock()

		if histogram, exists := mc.histograms.cache[metricKey]; exists {
			if err := validator.ValidateField("Labels", IsEmpty).Errors(); err != nil {
				raw, err := err.ToJSON()
				if err != nil {
					return err
				}
				return errors.New(string(raw))
			}
			histogram.WithLabelValues(metric.Labels...).Observe(metric.Histogram.ObservedValue)
			return nil
		}
		return fmt.Errorf("histogram not found: %v", metricKey)
	}

	if metric.Type == _GAUGE_ {
		mc.gauges.Lock()
		defer mc.gauges.Unlock()

		if gauge, exists := mc.gauges.cache[metricKey]; exists {
			if err := validator.ValidateField("Labels", IsEmpty).Errors(); err != nil {
				raw, err := err.ToJSON()
				if err != nil {
					return err
				}
				return errors.New(string(raw))
			}
			gauge.WithLabelValues(metric.Labels...).Set(metric.Gauge.Value)
			return nil
		}
		return fmt.Errorf("gauge not found: %v", metricKey)
	}

	if metric.Type == _SUMMARY_ {
		mc.summary.Lock()
		defer mc.summary.Unlock()

		if summary, exists := mc.summary.cache[metricKey]; exists {
			if err := validator.ValidateField("Labels", IsEmpty).Errors(); err != nil {
				raw, err := err.ToJSON()
				if err != nil {
					return err
				}
				return errors.New(string(raw))
			}
			summary.WithLabelValues(metric.Labels...)
			return nil
		}
		return fmt.Errorf("summary not found: %v", metricKey)
	}

	return fmt.Errorf("invalid metric type: %s", metric.Type)
}

func (mc *CollectorRegistry) register(metric MetricRequest) (prometheus.Collector, error) {
	metricKey := Metric{key: metric.Name}

	if metric.Type == _COUNTER_ {
		mc.counters.Lock()
		defer mc.counters.Unlock()

		if _, exists := mc.counters.cache[metricKey]; exists {
			return nil, fmt.Errorf("resource conflict: cannot reinitialize metric %v", metricKey)
		}

		counter := prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: metric.Name,
				Help: metric.Description,
			},
			metric.Labels,
		)
		mc.counters.cache[metricKey] = counter
		return counter, nil
	}

	if metric.Type == _HISTOGRAM_ {
		mc.histograms.Lock()
		defer mc.histograms.Unlock()

		if _, exists := mc.histograms.cache[metricKey]; exists {
			return nil, fmt.Errorf("resource conflict: cannot reinitialize metric %v", metricKey)
		}

		histogram := prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    metric.Name,
				Help:    metric.Description,
				Buckets: metric.Histogram.Buckets,
			},
			metric.Labels,
		)
		mc.histograms.cache[metricKey] = histogram
		return histogram, nil
	}

	if metric.Type == _GAUGE_ {
		mc.gauges.Lock()
		defer mc.gauges.Unlock()

		if _, exists := mc.gauges.cache[metricKey]; exists {
			return nil, fmt.Errorf("resource conflict: cannot reinitialize metric %v", metricKey)
		}

		gauge := prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: metric.Name,
				Help: metric.Description,
			},
			metric.Labels,
		)
		mc.gauges.cache[metricKey] = gauge
		return gauge, nil
	}

	if metric.Type == _SUMMARY_ {
		mc.summary.Lock()
		defer mc.summary.Unlock()

		if _, exists := mc.summary.cache[metricKey]; exists {
			return nil, fmt.Errorf("resource conflict: cannot reinitialize metric %v", metricKey)
		}

		if metric.Summary.MaxAge <= 0 {
			return nil, fmt.Errorf("invalid MaxAge for summary")
		}

		objectives := map[float64]float64{}
		for kobj, vobj := range metric.Summary.Objectives {
			v, err := strconv.ParseFloat(kobj, 64)
			if err != nil {
				return nil, fmt.Errorf("invalid objectives data: %v", err)
			}
			objectives[v] = vobj
		}

		summary := prometheus.NewSummaryVec(
			prometheus.SummaryOpts{
				Name:       metric.Name,
				Help:       metric.Description,
				Objectives: objectives,
				MaxAge:     time.Duration(metric.Summary.MaxAge),
			},
			metric.Labels,
		)
		mc.summary.cache[metricKey] = summary
		return summary, nil
	}

	return nil, fmt.Errorf("invalid metric type: %s", metric.Type)
}
