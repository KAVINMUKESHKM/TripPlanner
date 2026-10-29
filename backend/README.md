# Go backend

The Go API serves authentication, attraction search, nearby restaurant results, routing, and saved-trip operations. It uses PostgreSQL and loads `backend/.env` when run from this directory.

## Requirements

- Go 1.27.3 (see `go.mod`)
- PostgreSQL
- Optional provider API keys listed below

## Configure

Copy `.env.example` to `.env` only if `.env` does not already exist. Set `DATABASE_URL` to a PostgreSQL database the API can create tables in. Keep secrets out of source control.

| Variable | Required | Purpose |
| --- | --- | --- |
| `DATABASE_URL` | Yes | PostgreSQL connection string |
| `PORT` | No | API port; defaults to `8080` |
| `FRONTEND_URL` | Recommended | Allowed browser origin and OAuth return target; defaults to local frontend when configured |
| `OPENCAGE_API_KEY` | No | Geocode destination names missing from `destinations` |
| `OPENWEATHER_API_KEY` (`OPEN_WEATHER_KEY` also accepted) | No | Add daily weather summaries and match saved ideal-weather preferences |
| `SERP_API_KEY` (`SERP_KEY` also accepted) | No | Find nearby restaurants from attraction cards |
| `GOOGLE_CLIENT_ID` | No | Enables Google OAuth when paired with the secret |
| `GOOGLE_CLIENT_SECRET` | No | Enables Google OAuth when paired with the client ID |
| `GOOGLE_REDIRECT_URL` | No | OAuth callback, normally `http://localhost:8080/api/v1/auth/google/callback` |
| `COOKIE_SECURE` | No | Set `true` for HTTPS deployments; use `false` locally |
| `OSRM_BASE_URL` | No | OSRM endpoint; defaults to the public OSRM router |

## Run

From `backend/`:

```bash
go run ./cmd/migrate up
go run ./cmd/importosm                 # optional: import bundled Tamil Nadu data
go run ./cmd/importosm data/places.json # optional: import another compatible dataset
go run ./cmd/api
```

The migration command applies unapplied SQL migrations in filename order. The API does not run migrations automatically. The importer is safe to rerun for the same OpenStreetMap source IDs.

## API routes

All routes use the `/api/v1` prefix.

| Method | Path | Purpose |
| --- | --- | --- |
| `GET` | `/healthz` | Liveness check |
| `GET` | `/readyz` | Database readiness check |
| `POST` | `/auth/register` | Create an account and session |
| `POST` | `/auth/login` | Start an email/password session |
| `POST` | `/auth/logout` | End the current session |
| `GET` | `/auth/me` | Return the current signed-in user |
| `GET` | `/auth/google` | Start Google OAuth |
| `GET` | `/auth/google/callback` | Complete Google OAuth |
| `GET` | `/attractions?destination=…&days=…&date=YYYY-MM-DD&category=…` | Search and group attractions by itinerary day |
| `GET` | `/attractions/{id}` | Get one attraction |
| `GET` | `/attractions/{id}/restaurants` | Get nearby restaurants (SerpAPI key required) |
| `POST` | `/routes` | Get an OSRM route for ordered waypoints |
| `GET`, `POST` | `/trips` | List or create the signed-in user's trips |
| `GET`, `PUT`, `DELETE` | `/trips/{id}` | Read, update, or delete an owned trip |

Sessions use HttpOnly cookies. Trip routes require a signed-in user. For local Google sign-in, register the callback URL above in Google Cloud and set matching OAuth variables.

## Interactive API documentation

Start the API, then open [http://localhost:8080/docs](http://localhost:8080/docs) for Swagger UI. The OpenAPI 3.0 specification is served at [http://localhost:8080/openapi.yaml](http://localhost:8080/openapi.yaml). Swagger UI loads its interface assets from unpkg, so the interactive page needs an internet connection; the YAML specification is served directly by the Go API.

The frontend reads `NEXT_PUBLIC_OPENFREEMAP_STYLE_URL` for its MapLibre style; this is configured in `frontend/.env.local`, not in the backend.

## Development checks

```bash
go test ./...
go build ./...
```

The attraction dataset is OpenStreetMap data. Read [`data/README.md`](data/README.md) for attribution and license notes.
