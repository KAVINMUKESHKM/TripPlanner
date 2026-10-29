package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tripplanner/backend/internal/config"
)

type attraction struct {
	SourceID    string  `json:"source_id"`
	SourceURL   string  `json:"source_url"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Category    string  `json:"category"`
	Latitude    float64 `json:"latitude"`
	Longitude   float64 `json:"longitude"`
}

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) > 1 {
		return errors.New("usage: go run ./cmd/importosm [data-file]")
	}
	path := "data/tamil_nadu_attractions.json"
	if len(args) == 1 {
		path = args[0]
	}
	if err := config.LoadDotEnv(".env"); err != nil {
		return errors.New("could not load backend .env")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read attraction data: %w", err)
	}
	var attractions []attraction
	if err := json.Unmarshal(data, &attractions); err != nil {
		return fmt.Errorf("decode attraction data: %w", err)
	}
	if len(attractions) == 0 {
		return errors.New("attraction data is empty")
	}
	seen := make(map[string]struct{}, len(attractions))
	for _, place := range attractions {
		if place.SourceID == "" || !strings.HasPrefix(place.SourceURL, "https://www.openstreetmap.org/") || strings.TrimSpace(place.Name) == "" || place.Category == "" || place.Latitude < -90 || place.Latitude > 90 || place.Longitude < -180 || place.Longitude > 180 {
			return fmt.Errorf("invalid attraction record %q", place.SourceID)
		}
		if _, exists := seen[place.SourceID]; exists {
			return fmt.Errorf("duplicate OpenStreetMap ID %q", place.SourceID)
		}
		seen[place.SourceID] = struct{}{}
	}

	pool, err := pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		return errors.New("database configuration is invalid")
	}
	defer pool.Close()
	connectCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	if err := pool.Ping(connectCtx); err != nil {
		return errors.New("database is unavailable")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return errors.New("could not begin attraction import")
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	command, err := tx.Exec(ctx, `INSERT INTO attractions
		(source,source_id,source_url,name,description,category,latitude,longitude)
		SELECT 'OpenStreetMap',source_id,source_url,name,description,category,latitude,longitude
		FROM jsonb_to_recordset($1::jsonb) AS rows(
			source_id TEXT,source_url TEXT,name TEXT,description TEXT,category TEXT,
			latitude DOUBLE PRECISION,longitude DOUBLE PRECISION
		)
		ON CONFLICT (source,source_id) WHERE source IS NOT NULL AND source_id IS NOT NULL
		DO UPDATE SET source_url=EXCLUDED.source_url,name=EXCLUDED.name,description=EXCLUDED.description,
			category=EXCLUDED.category,latitude=EXCLUDED.latitude,longitude=EXCLUDED.longitude`, string(data))
	if err != nil {
		return fmt.Errorf("import attractions: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("could not commit attraction import")
	}
	fmt.Printf("Imported or updated %d Tamil Nadu attractions from OpenStreetMap.\n", command.RowsAffected())
	return nil
}
