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

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			c.Collect(ctx)
		}
	}
}

// Collect fetches and sets weather metrics.
func (c *Collector) Collect(ctx context.Context) {
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

	locationCount := len(resp.Records.Location)
	log.Printf("Successfully fetched data for %d locations", locationCount)

	for _, location := range resp.Records.Location {
		county := location.LocationName
		for _, element := range location.WeatherElement {
			for i, t := range element.Time {
				param := t.Parameter

				startTime := t.StartTime
				endTime := t.EndTime
				label := fmt.Sprintf(`county="%s", period="%d", start_time="%s", end_time="%s"`, county, i, startTime, endTime)

				switch element.ElementName {
				case "Wx":
					metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_weather_condition_info{%s, condition="%s"}`, label, param.ParameterName)).Set(1)
				case "PoP":
					if val, err := strconv.ParseFloat(param.ParameterName, 64); err == nil {
						metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_precipitation_probability_percent{%s}`, label)).Set(val)
					}
				case "MaxT":
					if val, err := strconv.ParseFloat(param.ParameterName, 64); err == nil {
						metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_temperature_celsius{%s, type="MaxT"}`, label)).Set(val)
					}
				case "MinT":
					if val, err := strconv.ParseFloat(param.ParameterName, 64); err == nil {
						metrics.GetOrCreateFloatCounter(fmt.Sprintf(`cwa_temperature_celsius{%s, type="MinT"}`, label)).Set(val)
					}
				}
			}
		}
	}

	metrics.GetOrCreateFloatCounter("cwa_last_fetch_success_timestamp_seconds").Set(float64(time.Now().Unix()))
}
