# Tamil Nadu attraction data

`tamil_nadu_attractions.json` contains 791 named tourism and historic features fetched from OpenStreetMap with the Overpass API on 2026-10-22. The import includes named attractions, museums, viewpoints, zoos, theme parks, aquariums, galleries, artwork, monuments, memorials, archaeological sites, forts, castles, and ruins within Tamil Nadu's administrative boundary. One duplicate name and coordinate was removed.

Each record retains its OpenStreetMap element URL. The data is available under the Open Database License (ODbL); see <https://www.openstreetmap.org/copyright>. The application displays OpenStreetMap attribution alongside imported place results.

From the `backend` directory, apply migrations and import or refresh the records with:

```sh
go run ./cmd/migrate up
go run ./cmd/importosm
```
