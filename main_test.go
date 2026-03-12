package main_test

import (
	"context"
	"os"
	"strings"
	"testing"
	"time"

	main "github.com/omegaatt36/cwa-exporter"
	"github.com/urfave/cli/v3"
)

func TestConfigFlags(t *testing.T) {
	originalAPIKey := os.Getenv("CWA_API_KEY")
	originalCounties := os.Getenv("CWA_COUNTIES")
	originalPollInterval := os.Getenv("CWA_POLL_INTERVAL")
	originalListenAddr := os.Getenv("LISTEN_ADDR")
	defer func() {
		os.Setenv("CWA_API_KEY", originalAPIKey)
		os.Setenv("CWA_COUNTIES", originalCounties)
		os.Setenv("CWA_POLL_INTERVAL", originalPollInterval)
		os.Setenv("LISTEN_ADDR", originalListenAddr)
	}()

	tests := []struct {
		name     string
		env      map[string]string
		args     []string
		validate func(*testing.T, *main.Config)
	}{
		{
			name: "default values",
			env: map[string]string{
				"CWA_API_KEY": "test-api",
			},
			validate: func(t *testing.T, cfg *main.Config) {
				if cfg.APIKey != "test-api" {
					t.Errorf("APIKey: expected test-api, got %s", cfg.APIKey)
				}
				if len(cfg.Counties) != 0 {
					t.Errorf("Counties: expected empty, got %v", cfg.Counties)
				}
				if cfg.PollInterval != 10*time.Minute {
					t.Errorf("PollInterval: expected 10m, got %v", cfg.PollInterval)
				}
				if cfg.ListenAddr != ":9100" {
					t.Errorf("ListenAddr: expected :9100, got %s", cfg.ListenAddr)
				}
			},
		},
		{
			name: "env mapping",
			env: map[string]string{
				"CWA_API_KEY":       "test-api-env",
				"CWA_COUNTIES":      "TaipeiCity,TaoyuanCity",
				"CWA_POLL_INTERVAL": "1h",
				"LISTEN_ADDR":       ":8080",
			},
			validate: func(t *testing.T, cfg *main.Config) {
				if cfg.APIKey != "test-api-env" {
					t.Errorf("APIKey: expected test-api-env, got %s", cfg.APIKey)
				}
				expectedCounties := []string{"TaipeiCity", "TaoyuanCity"}
				if len(cfg.Counties) != 2 || cfg.Counties[0] != expectedCounties[0] || cfg.Counties[1] != expectedCounties[1] {
					t.Errorf("Counties: expected %v, got %v", expectedCounties, cfg.Counties)
				}
				if cfg.PollInterval != time.Hour {
					t.Errorf("PollInterval: expected 1h, got %v", cfg.PollInterval)
				}
				if cfg.ListenAddr != ":8080" {
					t.Errorf("ListenAddr: expected :8080, got %s", cfg.ListenAddr)
				}
			},
		},
		{
			name: "cli flags override env",
			env: map[string]string{
				"CWA_API_KEY": "env-api",
			},
			args: []string{"--api-key", "cli-api", "--counties", "Hsinchu", "--poll-interval", "5m", "--listen-addr", ":7070"},
			validate: func(t *testing.T, cfg *main.Config) {
				if cfg.APIKey != "cli-api" {
					t.Errorf("APIKey: expected cli-api, got %s", cfg.APIKey)
				}
				if len(cfg.Counties) != 1 || cfg.Counties[0] != "Hsinchu" {
					t.Errorf("Counties: expected [Hsinchu], got %v", cfg.Counties)
				}
				if cfg.PollInterval != 5*time.Minute {
					t.Errorf("PollInterval: expected 5m, got %v", cfg.PollInterval)
				}
				if cfg.ListenAddr != ":7070" {
					t.Errorf("ListenAddr: expected :7070, got %s", cfg.ListenAddr)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Unsetenv("CWA_API_KEY")
			os.Unsetenv("CWA_COUNTIES")
			os.Unsetenv("CWA_POLL_INTERVAL")
			os.Unsetenv("LISTEN_ADDR")

			for k, v := range tt.env {
				os.Setenv(k, v)
			}

			cmd := main.NewAppCommand()
			var capturedCfg *main.Config

			originalAction := cmd.Action
			cmd.Action = func(ctx context.Context, c *cli.Command) error {
				counties := c.StringSlice("counties")
				var finalCounties []string
				for _, county := range counties {
					if strings.Contains(county, ",") {
						finalCounties = append(finalCounties, strings.Split(county, ",")...)
					} else {
						finalCounties = append(finalCounties, county)
					}
				}

				capturedCfg = &main.Config{
					APIKey:       c.String("api-key"),
					PollInterval: c.Duration("poll-interval"),
					ListenAddr:   c.String("listen-addr"),
				}
				capturedCfg.Counties = finalCounties
				return nil
			}
			_ = originalAction // keep compiler happy

			args := append([]string{"cwa-exporter"}, tt.args...)
			if err := cmd.Run(context.Background(), args); err != nil {
				t.Fatalf("cmd.Run failed: %v", err)
			}
			tt.validate(t, capturedCfg)
		})
	}
}

func TestRequiredAPIKey(t *testing.T) {
	os.Unsetenv("CWA_API_KEY")
	cmd := main.NewAppCommand()
	cmd.Action = func(ctx context.Context, c *cli.Command) error {
		return nil
	}

	err := cmd.Run(context.Background(), []string{"cwa-exporter"})
	if err == nil {
		t.Fatal("expected error for missing CWA_API_KEY, got nil")
	}
}
