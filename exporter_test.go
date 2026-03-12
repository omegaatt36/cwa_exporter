package main

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/omegaatt36/go-cwa"
)

func TestCollector_fetchAndSetMetrics(t *testing.T) {
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/rest/datastore/F-C0032-001", func(w http.ResponseWriter, r *http.Request) {
		resp := cwa.Response[cwa.Forecast36hRecords]{
			Success: "true",
			Records: cwa.Forecast36hRecords{
				Location: []cwa.Location{
					{
						LocationName: "Taipei",
						WeatherElement: []cwa.WeatherElement{
							{
								ElementName: "Wx",
								Time: []cwa.Time{
									{
										Parameter: cwa.Parameter{ParameterName: "Cloudy"},
									},
								},
							},
							{
								ElementName: "PoP",
								Time: []cwa.Time{
									{
										Parameter: cwa.Parameter{ParameterName: "20"},
									},
								},
							},
							{
								ElementName: "MaxT",
								Time: []cwa.Time{
									{
										Parameter: cwa.Parameter{ParameterName: "25"},
									},
								},
							},
							{
								ElementName: "MinT",
								Time: []cwa.Time{
									{
										Parameter: cwa.Parameter{ParameterName: "18"},
									},
								},
							},
						},
					},
				},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	server := httptest.NewServer(mux)
	defer server.Close()

	client := cwa.NewClient("fake-key", cwa.WithBaseURL(server.URL+"/"))
	config := &Config{
		Counties:     []string{"Taipei"},
		PollInterval: 30 * time.Minute,
	}

	collector := NewCollector(client, config)
	collector.Collect(context.Background())

	// Verify metrics
	var buf bytes.Buffer
	metrics.WritePrometheus(&buf, false)
	output := buf.String()

	expectedMetrics := []string{
		`cwa_weather_condition_info{county="Taipei", period="0", condition="Cloudy"} 1`,
		`cwa_precipitation_probability_percent{county="Taipei", period="0"} 20`,
		`cwa_temperature_celsius{county="Taipei", period="0", type="MaxT"} 25`,
		`cwa_temperature_celsius{county="Taipei", period="0", type="MinT"} 18`,
		`cwa_last_fetch_success_timestamp_seconds`,
	}

	for _, m := range expectedMetrics {
		if !strings.Contains(output, m) {
			t.Errorf("Expected metric %q not found in output:\n%s", m, output)
		}
	}
}
