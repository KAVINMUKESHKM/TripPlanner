# Next.js frontend

The frontend is a Next.js App Router application using React, TypeScript, Material UI, MapLibre, and the Go API.

## Requirements

- Node.js 20 or later
- npm
- Go API available at `http://localhost:8080` for full functionality

## Configure and run

```bash
npm install
cp .env.example .env.local   # only if .env.local does not already exist
npm run dev
```

Open [http://localhost:3000](http://localhost:3000). `next.config.ts` proxies `/api/*` requests to the server specified by `API_INTERNAL_URL`, which defaults to `http://localhost:8080`.

| Variable | Purpose |
| --- | --- |
| `API_INTERNAL_URL` | Server-side API proxy target |
| `NEXT_PUBLIC_API_BASE_URL` | Optional browser-visible API base URL; leave blank for same-origin proxying |
| `NEXT_PUBLIC_OPENFREEMAP_STYLE_URL` | Optional MapLibre style URL; defaults to OpenFreeMap Liberty |

Google sign-in is configured on the backend. The registered OAuth callback must return the browser to this frontend.

## Pages and behavior

- `/` — destination, date, and duration search, with trip ideas.
- `/plan` — attraction search, weather and category filtering, day selection, attraction details, nearby restaurants, editable itinerary, and map route summary.
- `/login` and `/signup` — email/password and Google sign-in entry.
- `/auth/callback` — completes Google sign-in.
- `/trips` — list and delete saved trips.
- `/trips/{id}` — edit a saved itinerary.

Nearby restaurant lookup is optional. If the backend reports that it is not configured, add `SERP_API_KEY` (or keep the legacy `SERP_KEY`) to `backend/.env`. Restaurant map links open Google Maps; the itinerary map itself uses MapLibre with OpenFreeMap tiles and OSRM routing.

## Build

```bash
npm run build
npm run start
```
