package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"tripplanner/backend/internal/config"
)

const defaultOverpassURL = "https://overpass.private.coffee/api/interpreter"

type restaurant struct {
	SourceID     string  `json:"source_id"`
	SourceURL    string  `json:"source_url"`
	Name         string  `json:"name"`
	Amenity      string  `json:"amenity"`
	Cuisine      string  `json:"cuisine"`
	Address      string  `json:"address"`
	OpeningHours string  `json:"opening_hours"`
	Phone        string  `json:"phone"`
	Website      string  `json:"website"`
	Latitude     float64 `json:"latitude"`
	Longitude    float64 `json:"longitude"`
}

type overpassResponse struct {
	Elements []struct {
		Type   string  `json:"type"`
		ID     int64   `json:"id"`
		Lat    float64 `json:"lat"`
		Lon    float64 `json:"lon"`
		Center *struct {
			Lat float64 `json:"lat"`
			Lon float64 `json:"lon"`
		} `json:"center"`
		Tags map[string]string `json:"tags"`
	} `json:"elements"`
}

func main() {
	if err := run(context.Background()); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context) error {
	if err := config.LoadDotEnv(".env"); err != nil {
		return errors.New("could not load backend .env")
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	endpoint := strings.TrimSpace(os.Getenv("OVERPASS_URL"))
	if endpoint == "" {
		endpoint = defaultOverpassURL
	}
	requestCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
	defer cancel()
	query := `[out:json][timeout:180];area["name"="Tamil Nadu"]["admin_level"="4"]["boundary"="administrative"]->.state;(nwr["amenity"~"^(restaurant|cafe|fast_food|food_court|ice_cream|pub|bar)$"]["name"](area.state););out center tags;`
	req, err := http.NewRequestWithContext(requestCtx, http.MethodPost, endpoint, strings.NewReader(url.Values{"data": []string{query}}.Encode()))
	if err != nil {
		return fmt.Errorf("create Overpass request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("User-Agent", "TripPlanner OSM restaurant importer")
	resp, err := (&http.Client{Timeout: 3 * time.Minute}).Do(req)
	if err != nil {
		return fmt.Errorf("request OpenStreetMap restaurants: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("Overpass returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 64<<20))
	if err != nil {
		return fmt.Errorf("read Overpass response: %w", err)
	}
	var result overpassResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return fmt.Errorf("decode Overpass response: %w", err)
	}
	places := make([]restaurant, 0, len(result.Elements))
	seen := make(map[string]struct{}, len(result.Elements))
	for _, element := range result.Elements {
		if element.Type == "" || element.ID <= 0 || element.Tags["name"] == "" {
			continue
		}
		lat, lon := element.Lat, element.Lon
		if element.Center != nil {
			lat, lon = element.Center.Lat, element.Center.Lon
		}
		if lat < -90 || lat > 90 || lon < -180 || lon > 180 {
			continue
		}
		id := element.Type + "/" + strconv.FormatInt(element.ID, 10)
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		tags := element.Tags
		places = append(places, restaurant{
			SourceID: id, SourceURL: "https://www.openstreetmap.org/" + id,
			Name: strings.TrimSpace(tags["name"]), Amenity: tags["amenity"], Cuisine: tags["cuisine"],
			Address: addressFromTags(tags), OpeningHours: tags["opening_hours"],
			Phone: firstTag(tags, "phone", "contact:phone"), Website: firstTag(tags, "website", "contact:website", "url"),
			Latitude: lat, Longitude: lon,
		})
	}
	if len(places) == 0 {
		return errors.New("Overpass returned no named food places for Tamil Nadu")
	}
	data, err := json.Marshal(places)
	if err != nil {
		return fmt.Errorf("encode restaurant data: %w", err)
	}
	databaseURL := cfg.DatabaseURL
	if cfg.ImporterDatabaseURL != "" {
		databaseURL = cfg.ImporterDatabaseURL
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		return errors.New("database configuration is invalid")
	}
	defer pool.Close()
	connectCtx, connectCancel := context.WithTimeout(ctx, 10*time.Second)
	defer connectCancel()
	if err := pool.Ping(connectCtx); err != nil {
		return errors.New("database is unavailable")
	}
	tx, err := pool.Begin(ctx)
	if err != nil {
		return errors.New("could not begin restaurant import")
	}
	defer func() { _ = tx.Rollback(context.Background()) }()
	command, err := tx.Exec(ctx, `INSERT INTO restaurants
		(source_id,source_url,name,amenity,cuisine,address,opening_hours,phone,website,location)
		SELECT source_id,source_url,name,amenity,cuisine,NULLIF(address,''),NULLIF(opening_hours,''),NULLIF(phone,''),NULLIF(website,''),
			ST_SetSRID(ST_MakePoint(longitude,latitude),4326)::geography
		FROM jsonb_to_recordset($1::jsonb) AS rows(
			source_id TEXT,source_url TEXT,name TEXT,amenity TEXT,cuisine TEXT,address TEXT,
			opening_hours TEXT,phone TEXT,website TEXT,latitude DOUBLE PRECISION,longitude DOUBLE PRECISION
		)
		ON CONFLICT (source_id) DO UPDATE SET source_url=EXCLUDED.source_url,name=EXCLUDED.name,
			amenity=EXCLUDED.amenity,cuisine=EXCLUDED.cuisine,address=EXCLUDED.address,
			opening_hours=EXCLUDED.opening_hours,phone=EXCLUDED.phone,website=EXCLUDED.website,location=EXCLUDED.location`, string(data))
	if err != nil {
		return fmt.Errorf("import OSM restaurants: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return errors.New("could not commit restaurant import")
	}
	fmt.Printf("Imported or updated %d OpenStreetMap food places into PostGIS.\n", command.RowsAffected())
	return nil
}

func addressFromTags(tags map[string]string) string {
	if value := strings.TrimSpace(tags["addr:full"]); value != "" {
		return value
	}
	parts := make([]string, 0, 4)
	for _, key := range []string{"addr:housenumber", "addr:street", "addr:suburb", "addr:city"} {
		if value := strings.TrimSpace(tags[key]); value != "" {
			parts = append(parts, value)
		}
	}
	return strings.Join(parts, ", ")
}

func firstTag(tags map[string]string, keys ...string) string {
	for _, key := range keys {
		if value := strings.TrimSpace(tags[key]); value != "" {
			return value
		}
	}
	return ""
}
