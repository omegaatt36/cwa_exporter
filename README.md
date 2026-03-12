# CWA Weather Exporter

A lightweight Prometheus exporter for Taiwan Central Weather Administration (CWA) Open Data.

## Features

- **Lightweight:** Built with `VictoriaMetrics/metrics` for a minimal footprint.
- **Efficient:** Uses a background poller to respect API rate limits and ensure fast Prometheus scrapes.
- **Configurable:** Filter by specific counties and adjust polling intervals via environment variables.
- **Secure:** Packaged using Google's `distroless` static image.

## Configuration

The exporter is configured entirely through environment variables:

| Variable | Description | Default |
|----------|-------------|---------|
| `CWA_API_KEY` | **Required.** Your CWA Open Data API Key. | - |
| `CWA_COUNTIES` | Comma-separated list of counties (e.g., `TaipeiCity,TaoyuanCity`). | All counties |
| `CWA_POLL_INTERVAL` | How often to fetch data (e.g., `30m`, `1h`). | `30m` |
| `LISTEN_ADDR` | Address to listen on for metrics. | `:9100` |

## Metrics

- `cwa_temperature_celsius`: Forecast temperature (MaxT/MinT) by county.
- `cwa_precipitation_probability_percent`: Probability of precipitation by county.
- `cwa_weather_condition_info`: Current weather condition description (Gauge set to 1).
- `cwa_last_fetch_success_timestamp_seconds`: Timestamp of the last successful API fetch.

## Usage

### Local

```bash
export CWA_API_KEY="your-api-key"
go run .
```

### Docker

```bash
docker build -t cwa-exporter .
docker run -e CWA_API_KEY="your-api-key" -p 9100:9100 cwa-exporter
```

Access metrics at `http://localhost:9100/metrics`.

## License

MIT
