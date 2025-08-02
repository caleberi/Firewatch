package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"

	"github.com/prometheus/client_golang/prometheus"
)

func RegisterRestMetric(mc *CollectorRegistry, reg *prometheus.Registry) http.HandlerFunc {
	return http.HandlerFunc(
		func(res http.ResponseWriter, req *http.Request) {
			var metricReq MetricRequest

			err := parseRequestBody(req, &metricReq)
			if err != nil {
				response := MetricResponse{
					Status: http.StatusBadRequest,
					Reason: fmt.Sprintf("failed to parse request body: %v", err),
				}
				raw, err := response.ToJSON()
				if err != nil {
					http.Error(res, fmt.Sprintf("failed to marshal error response: %v", err), http.StatusInternalServerError)
					return
				}
				res.WriteHeader(http.StatusBadRequest)
				res.Header().Set("Content-Type", "application/json")
				res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
				res.Write(raw)
				return
			}

			validator := NewValidator(metricReq)
			validationErr := validator.
				ValidateField("Name", IsEmpty).
				ValidateField("Type", IsEmpty,
					IsSupported(_COUNTER_, _GAUGE_, _HISTOGRAM_, _SUMMARY_)).
				Errors()

			if validationErr != nil {
				raw, err := validationErr.ToJSON()
				if err != nil {
					http.Error(res, err.Error(), http.StatusBadRequest)
					return
				}
				response := MetricResponse{
					Status: http.StatusBadRequest,
					Reason: string(raw),
				}
				raw, err = response.ToJSON()
				if err != nil {
					http.Error(res, fmt.Sprintf("failed to marshal validation error response: %v", err), http.StatusInternalServerError)
					return
				}
				res.WriteHeader(http.StatusBadRequest)
				res.Header().Set("Content-Type", "application/json")
				res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
				res.Write(raw)
				return
			}

			collector, err := mc.register(metricReq)
			if err != nil {
				response := MetricResponse{
					Status: http.StatusBadRequest,
					Reason: err.Error(),
				}
				raw, err := response.ToJSON()
				if err != nil {
					http.Error(res, fmt.Sprintf("failed to marshal error response: %v", err), http.StatusInternalServerError)
					return
				}
				res.WriteHeader(http.StatusBadRequest)
				res.Header().Set("Content-Type", "application/json")
				res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
				res.Write(raw)
				return
			}

			if err = reg.Register(collector); err != nil {
				response := MetricResponse{
					Status: http.StatusInternalServerError,
					Reason: fmt.Sprintf("failed to register metric: %v", err),
				}
				raw, err := response.ToJSON()
				if err != nil {
					http.Error(res, fmt.Sprintf("failed to marshal error response: %v", err), http.StatusInternalServerError)
					return
				}
				res.WriteHeader(http.StatusInternalServerError)
				res.Header().Set("Content-Type", "application/json")
				res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
				res.Write(raw)
				return
			}

			response := MetricResponse{
				Status:  http.StatusCreated,
				Message: fmt.Sprintf("Metric %s created successfully", metricReq.Name),
			}
			raw, err := response.ToJSON()
			if err != nil {
				http.Error(res, fmt.Sprintf("failed to marshal success response: %v", err), http.StatusInternalServerError)
				return
			}
			res.WriteHeader(http.StatusCreated)
			res.Header().Set("Content-Type", "application/json")
			res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
			res.Write(raw)
		})
}

func PushStatRestMetric(mc *CollectorRegistry, reg *prometheus.Registry) http.HandlerFunc {
	return http.HandlerFunc(func(res http.ResponseWriter, req *http.Request) {
		var metricReq MetricRequest

		err := parseRequestBody(req, &metricReq)
		if err != nil {
			response := MetricResponse{
				Status: http.StatusBadRequest,
				Reason: fmt.Sprintf("failed to parse request body: %v", err),
			}
			raw, err := response.ToJSON()
			if err != nil {
				http.Error(res, fmt.Sprintf("failed to marshal error response: %v", err), http.StatusInternalServerError)
				return
			}
			res.WriteHeader(http.StatusBadRequest)
			res.Header().Set("Content-Type", "application/json")
			res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
			res.Write(raw)
			return
		}

		validator := NewValidator(metricReq)
		validationErr := validator.
			ValidateField("Type", IsEmpty).
			ValidateField("Name", IsEmpty).Errors()

		if validationErr != nil {
			raw, err := validationErr.ToJSON()
			if err != nil {
				http.Error(res, err.Error(), http.StatusBadRequest)
				return
			}
			response := MetricResponse{
				Status: http.StatusBadRequest,
				Reason: string(raw),
			}
			raw, err = response.ToJSON()
			if err != nil {
				http.Error(res, fmt.Sprintf("failed to marshal validation error response: %v", err), http.StatusInternalServerError)
				return
			}
			res.WriteHeader(http.StatusBadRequest)
			res.Header().Set("Content-Type", "application/json")
			res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
			res.Write(raw)
			return
		}

		if err = mc.update(metricReq); err != nil {
			response := MetricResponse{
				Status: http.StatusBadRequest,
				Reason: err.Error(),
			}
			raw, err := response.ToJSON()
			if err != nil {
				http.Error(res, fmt.Sprintf("failed to marshal error response: %v", err), http.StatusInternalServerError)
				return
			}
			res.WriteHeader(http.StatusBadRequest)
			res.Header().Set("Content-Type", "application/json")
			res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
			res.Write(raw)
			return
		}

		response := MetricResponse{
			Status:  http.StatusOK,
			Message: fmt.Sprintf("Metric %s updated successfully", metricReq.Name),
		}
		raw, err := response.ToJSON()
		if err != nil {
			http.Error(res, fmt.Sprintf("failed to marshal success response: %v", err), http.StatusInternalServerError)
			return
		}
		res.WriteHeader(http.StatusOK)
		res.Header().Set("Content-Type", "application/json")
		res.Header().Set("Content-Length", strconv.FormatInt(int64(len(raw)), 10))
		res.Write(raw)
	})
}

func parseRequestBody(req *http.Request, metricReq *MetricRequest) error {
	data, err := io.ReadAll(req.Body)
	if err != nil {
		return fmt.Errorf("failed to read request body: (%s)", err)
	}

	if err := json.Unmarshal(data, metricReq); err != nil {
		return fmt.Errorf("invalid JSON format: (%s)", err)
	}

	return nil
}
