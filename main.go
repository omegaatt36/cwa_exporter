package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/VictoriaMetrics/metrics"
	"github.com/omegaatt36/go-cwa"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := NewAppCommand()
	if err := cmd.Run(context.Background(), os.Args); err != nil {
		os.Exit(1)
	}
}

// Config represents the application configuration.
type Config struct {
	APIKey       string
	Counties     []string
	PollInterval time.Duration
	ListenAddr   string
}

// NewAppCommand creates a new CLI command for the CWA Exporter.
func NewAppCommand() *cli.Command {
	cfg := &Config{}
	return &cli.Command{
		Name:  "cwa-exporter",
		Usage: "Export CWA weather data as Prometheus metrics",
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:        "api-key",
				Usage:       "CWA API Key",
				Sources:     cli.EnvVars("CWA_API_KEY"),
				Required:    true,
				Destination: &cfg.APIKey,
			},
			&cli.StringSliceFlag{
				Name:    "counties",
				Usage:   "Counties to poll (can be specified multiple times or as comma-separated in CWA_COUNTIES)",
				Sources: cli.EnvVars("CWA_COUNTIES"),
			},
			&cli.DurationFlag{
				Name:        "poll-interval",
				Usage:       "Interval between polls",
				Sources:     cli.EnvVars("CWA_POLL_INTERVAL"),
				Value:       10 * time.Minute,
				Destination: &cfg.PollInterval,
			},
			&cli.StringFlag{
				Name:        "listen-addr",
				Usage:       "Address to listen on for metrics",
				Sources:     cli.EnvVars("LISTEN_ADDR"),
				Value:       ":9100",
				Destination: &cfg.ListenAddr,
			},
		},
		Action: func(ctx context.Context, cmd *cli.Command) error {
			counties := cmd.StringSlice("counties")
			var finalCounties []string
			for _, c := range counties {
				if strings.Contains(c, ",") {
					finalCounties = append(finalCounties, strings.Split(c, ",")...)
				} else {
					finalCounties = append(finalCounties, c)
				}
			}
			cfg.Counties = finalCounties
			return runExporter(ctx, cfg)
		},
	}
}

func runExporter(ctx context.Context, config *Config) error {
	client := cwa.NewClient(config.APIKey)
	collector := NewCollector(client, config)

	log.Println("Performing initial data fetch...")
	collector.Collect(ctx)

	go collector.StartPoller(ctx)

	mux := http.NewServeMux()
	mux.HandleFunc("GET /metrics", func(w http.ResponseWriter, r *http.Request) {
		metrics.WritePrometheus(w, false)
	})
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	})

	server := &http.Server{
		Addr:    config.ListenAddr,
		Handler: mux,
	}

	go func() {
		log.Printf("Starting server on %s", config.ListenAddr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("Server error: %v", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case sig := <-sigChan:
		log.Printf("Received signal %v, shutting down...", sig)
	case <-ctx.Done():
		log.Printf("Context cancelled, shutting down...")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	return server.Shutdown(shutdownCtx)
}
