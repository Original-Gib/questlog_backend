# CLAUDE.md

This file provides guidance to Claude Code when working with code in this repository.

## Project Overview

QuestLog backend — a Go REST API serving the Vue 3 frontend at `../questlog_frontend`. The API provides endpoints for game tracking: user auth, game/console search, library management (status tracking), wishlists, reviews, and analytics.

## Commands

```bash
go run main.go       # Start dev server on :8080
go build             # Build binary
go test ./...        # Run all tests
go get <pkg>         # Add a dependency
```

## Architecture

Go + Gin REST API backend acting as a middleware layer between the Vue frontend and external services.

```
Vue frontend → Go/Gin API (:8080) → IGDB (game/console data)
                                   → Supabase (user data)
```

- **Entry**: `main.go` — bootstraps Gin router and starts server on `:8080`
- **Module**: `github.com/Original_Gib/questlog`
- **Framework**: [Gin](https://github.com/gin-gonic/gin)

## External Integrations

- **IGDB** — game and console metadata (search, cover art, etc.); uses Twitch OAuth (`TWITCH_CLIENT_ID`, `TWITCH_CLIENT_SECRET`)
- **Supabase** — primary data store for all user data (library, wishlists, reviews, auth); uses `SUPABASE_URL` and `SUPABASE_KEY`

## Frontend

The Vue 3 frontend lives at `../questlog_frontend` and communicates with this API via `VITE_API_BASE_URL` (defaults to `http://localhost:8080`). See `../questlog_frontend/CLAUDE.md` for frontend architecture details.

## Code Style

- Standard Go conventions (`gofmt`, idiomatic Go)
- Return JSON responses using `c.JSON()` with `gin.H{}` or typed structs
