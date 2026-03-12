package main

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/omegaatt36/go-cwa"
)

// Collector wraps CWA client and configuration to fetch and export metrics.
type Collector struct {
	client *cwa.Client
	config *Config
}

// NewCollector creates a new Collector instance.
func NewCollector(client *cwa.Client, config *Config) *Collector {
	return &Collector{
		client: client,
		config: config,
	}
}

// StartPoller starts a background goroutine to poll CWA data periodically.
func (c *Collector) StartPoller(ctx context.Context) {
	ticker := time.NewTicker(c.config.PollInterval)
	defer ticker.Stop()

	c.fetchAndSetMetrics(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.fetchAndSetMetrics(ctx)
		}
	}
}

func (c *Collector) fetchAndSetMetrics(ctx context.Context) {
	params := &cwa.Forecast36hParams{}
	if len(c.config.Counties) > 0 {
		counties := make([]cwa.County, len(c.config.Counties))
		for i, v := range c.config.Counties {
			counties[i] = cwa.County(v)
		}
		params.LocationNames = counties
	}

	resp, err := c.client.Get36hForecast(ctx, params)
	if err != nil {
		log.Printf("Error fetching weather data: %v", err)
		return
	}

	if resp == nil || !resp.IsSuccess() {
		log.Printf("CWA API request failed or returned no data")
		return
	}

	for _, location := range resp.Records.Location {
		county := location.LocationName
		for _, element := range location.WeatherElement {
			if len(element.Time) == 0 {
				continue
			}
			// Use the first forecast period (usually the current or next one)
			t := element.Time[0]
			param := t.Parameter

			switch element.ElementName {
			case "Wx":
				// cwa_weather_condition_info{county="...", condition="..."}
				metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_weather_condition_info{county="%s", condition="%s"}`, county, param.ParameterName)).Set(1)
			case "PoP":
				// cwa_precipitation_probability_percent{county="..."}
				if val, err := strconv.ParseFloat(param.ParameterName, 64); err == nil {
					metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_precipitation_probability_percent{county="%s"}`, county)).Set(val)
				}
			case "MaxT":
				// cwa_temperature_celsius{county="...", type="MaxT"}
				if val, err := strconv.ParseFloat(param.ParameterName, 64); err == nil {
					metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_temperature_celsius{county="%s", type="MaxT"}`, county)).Set(val)
				}
			case "MinT":
				// cwa_temperature_celsius{county="...", type="MinT"}
				if val, err := strconv.ParseFloat(param.ParameterName, 64); err == nil {
					metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_temperature_celsius{county="%s", type="MinT"}`, county)).Set(val)
				}
			}
		}
	}

	metrics.GetOrCreateFloatCounter("cwa_last_fetch_success_timestamp_seconds").Set(float64(time.Now().Unix()))
}
