package config_test

import (
	"testing"

	"tripplanner/backend/internal/config"
)

func TestLoadRequiresDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "")
	if _, err := config.Load(); err == nil {
		t.Fatal("Load() should reject a missing DATABASE_URL")
	}
}

func TestLoadReadsDatabaseURLAndOptionalProviderKeys(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("OPENWEATHER_API_KEY", "weather-key")
	t.Setenv("OPENCAGE_API_KEY", "geo-key")
	t.Setenv("OSRM_BASE_URL", "https://router.project-osrm.org")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.DatabaseURL != "postgres://example" || cfg.OpenWeatherAPIKey != "weather-key" || cfg.OpenCageAPIKey != "geo-key" || cfg.OSRMBaseURL != "https://router.project-osrm.org" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestLoadAcceptsLegacyWeatherKeyName(t *testing.T) {
	t.Setenv("DATABASE_URL", "postgres://example")
	t.Setenv("OPENWEATHER_API_KEY", "")
	t.Setenv("OPEN_WEATHER_KEY", "legacy-weather")
	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if cfg.OpenWeatherAPIKey != "legacy-weather" {
		t.Fatalf("legacy keys were not loaded: %+v", cfg)
	}
}
