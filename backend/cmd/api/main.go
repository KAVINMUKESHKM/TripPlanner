package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/oauth2"
	"tripplanner/backend/internal/auth"
	"tripplanner/backend/internal/config"
	"tripplanner/backend/internal/httpapi"
	"tripplanner/backend/internal/places"
	"tripplanner/backend/internal/providers"
	"tripplanner/backend/internal/trips"
)

func main() {
	if err := config.LoadDotEnv(".env"); err != nil {
		log.Fatalf("could not load backend .env: %v", err)
	}
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("invalid backend configuration: %v", err)
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("could not parse DATABASE_URL: %v", err)
	}
	poolConfig.MaxConns = 30
	poolConfig.MinConns = 4
	poolConfig.MaxConnLifetime = 1 * time.Hour
	poolConfig.MaxConnIdleTime = 15 * time.Minute
	poolConfig.HealthCheckPeriod = 1 * time.Minute

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		log.Fatalf("could not configure database connection (check DATABASE_URL): %v", err)
	}
	defer pool.Close()

	dbCtx, dbCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer dbCancel()
	if err := pool.Ping(dbCtx); err != nil {
		log.Fatalf("database connection failed (check DATABASE_URL and run migrations): %v", err)
	}
	log.Printf("database connection ready")

	appCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go cleanupExpiredSecurityRecords(appCtx, pool)

	var googleOAuth *oauth2.Config
	var googleVerifier auth.IdentityVerifier
	if cfg.GoogleClientID != "" && cfg.GoogleClientSecret != "" {
		redirectURL := cfg.GoogleRedirectURL
		if redirectURL == "" {
			redirectURL = strings.TrimRight(cfg.FrontendURL, "/") + "/api/v1/auth/google/callback"
		}
		log.Printf("Google OAuth callback URI: %s", redirectURL)
		googleOAuth = &oauth2.Config{
			ClientID:     cfg.GoogleClientID,
			ClientSecret: cfg.GoogleClientSecret,
			RedirectURL:  redirectURL,
			Endpoint: oauth2.Endpoint{
				AuthURL:  "https://accounts.google.com/o/oauth2/v2/auth",
				TokenURL: "https://oauth2.googleapis.com/token",
			},
			Scopes: []string{"openid", "email", "profile"},
		}
		googleVerifier = auth.NewGoogleOIDCVerifier()
	}
	authHandler := auth.NewHandler(pool, auth.Options{
		CookieSecure:   cfg.CookieSecure,
		AllowedOrigin:  cfg.FrontendURL,
		GoogleOAuth:    googleOAuth,
		GoogleVerifier: googleVerifier,
		FrontendURL:    cfg.FrontendURL,
		SMTP: auth.SMTPConfig{
			Host:     cfg.SMTPHost,
			Port:     cfg.SMTPPort,
			Username: cfg.SMTPUsername,
			Password: cfg.SMTPPassword,
			From:     cfg.SMTPFrom,
		},
	})
	placeRepository := places.NewRepository(pool)
	placeHandler := places.NewHandler(placeRepository, providers.Geocoder{APIKey: cfg.OpenCageAPIKey}, providers.Weather{APIKey: cfg.OpenWeatherAPIKey}, providers.Router{BaseURL: cfg.OSRMBaseURL}, placeRepository)
	tripHandler := trips.NewHandler(trips.NewRepository(pool), authHandler)
	rateLimiter, err := httpapi.NewPostgresRateLimiter(pool, cfg.TrustedProxies)
	if err != nil {
		log.Fatal(err)
	}
	log.Printf("API starting: address=:%s environment=%s frontend=%s", cfg.Port, valueOr(cfg.Environment, "development"), valueOr(cfg.FrontendURL, "not configured"))
	server := &http.Server{
		Addr:              ":" + cfg.Port,
		Handler:           httpapi.NewRouter(httpapi.Dependencies{Ready: pool.Ping, Limiter: rateLimiter, Auth: authHandler.Routes(), Places: placeHandler.Routes(), Trips: tripHandler.Routes()}),
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
		MaxHeaderBytes:    1 << 20,
	}

	serverErr := make(chan error, 1)
	go func() {
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			serverErr <- err
		}
	}()

	select {
	case err := <-serverErr:
		log.Fatalf("server failed: %v", err)
	case <-appCtx.Done():
		log.Printf("shutdown signal received; draining in-flight requests...")
	}

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("graceful server shutdown failed: %v", err)
	}
	log.Printf("server stopped cleanly")
}

func valueOr(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func cleanupExpiredSecurityRecords(ctx context.Context, pool *pgxpool.Pool) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			cleanCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if _, err := pool.Exec(cleanCtx, "DELETE FROM sessions WHERE expires_at <= now()"); err != nil {
				log.Printf("cleanup sessions: %v", err)
			}
			if _, err := pool.Exec(cleanCtx, "DELETE FROM oauth_states WHERE expires_at <= now()"); err != nil {
				log.Printf("cleanup oauth states: %v", err)
			}
			if _, err := pool.Exec(cleanCtx, "DELETE FROM password_reset_tokens WHERE expires_at <= now()"); err != nil {
				log.Printf("cleanup password tokens: %v", err)
			}
			if _, err := pool.Exec(cleanCtx, "DELETE FROM api_rate_limits WHERE window_started <= now() - interval '5 minutes'"); err != nil {
				log.Printf("cleanup rate limits: %v", err)
			}
			cancel()
		}
	}
}
