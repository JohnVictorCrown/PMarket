# PMarket — Polymarket Copy Trading Platform

Go-based system that scans top Polymarket wallets, copies their trades, and shows results in a Svelte 5 dashboard.

## Components

| Component | Path | Description |
|-----------|------|-------------|
| **Scanner** | `scanner/` | Fetches Polymarket leaderboard & market data, scores wallets by PnL/Volume, generates HTML report |
| **Service** | `service/` | Copy-trading backend — polls tracked wallets for new/closed positions, places real or paper trades, exposes HTTP API |
| **Frontend** | `frontend/` | Svelte 5 dashboard built with Bun — shows P&L, open positions, scan metrics |

## Build & Run

### Scanner

```bash
cd scanner
go run ./cmd/scanner
```

### Service

```bash
cd service
go run ./cmd/server
```

Configure via env vars (see `service/pkg/config/config.go`). Wallets file at `wallets.config` by default.

### Frontend

```bash
cd frontend
bun run build    # production build — outputs to dist/
bun run dev      # watch mode (JS + Tailwind)
bun run preview  # serve dist/ locally via npx serve
```

API URL is hardcoded in `src/api.js` (`https://pmarket-vgtl.onrender.com`).

## Commands

| Command | Description |
|---------|-------------|
| `go run ./cmd/scanner` | Run scanner and generate report |
| `go run ./cmd/server` | Start copy-trading service |
| `cd frontend && bun run build` | Build frontend |
| `cd frontend && bun run preview` | Preview built frontend |
| `go test ./...` | Run Go tests |
| `go vet ./...` | Static analysis |
| `golangci-lint run ./...` | Full lint |

## Project Structure

```
├── scanner/
│   ├── cmd/scanner/main.go          — entry point
│   └── pkg/
│       ├── config/config.go         — env-based config for scanner
│       ├── polymarket/client.go     — read-only SDK wrapper (leaderboard, markets, closed positions)
│       ├── scanner/scanner.go       — orchestrates scan, scores & ranks wallets
│       ├── report/report.go         — HTML report generation (embed.FS templates)
│       ├── report/templates/report.html  — sortable HTML report template
│       └── types/types.go           — Wallet, MarketSummary, ScanResult
├── service/
│   ├── cmd/server/main.go           — entry point
│   └── pkg/
│       ├── config/config.go         — env + wallets.config loading
│       ├── polymarket/client.go     — authenticated SDK wrapper (orders, positions)
│       ├── tracker/tracker.go       — periodic wallet scanner, detects new/closed trades
│       ├── store/store.go           — persisted state (seen trades, open, copied) as JSON
│       ├── paper/paper.go           — paper trading with virtual balance
│       └── server/server.go         — HTTP API: /health, /uptime, /status, /balance
├── frontend/
│   ├── src/
│   │   ├── main.js                  — Svelte 5 entry, mounts App
│   │   ├── App.svelte               — dashboard UI with $state() runes
│   │   ├── api.js                   — API client with hardcoded BASE
│   │   └── app.css                  — @tailwind directives
│   ├── dist/                        — production build output
│   │   ├── index.html
│   │   ├── main.js                  — bundled JS (via Bun.build + SveltePlugin)
│   │   └── app.css                  — Tailwind-generated CSS (minified)
│   ├── index.html                   — dev entry (not used in production)
│   ├── build.js                     — Bun build script (clean, Tailwind CLI, Bun.build, write HTML)
│   ├── package.json                 — svelte, tailwindcss, @skeletonlabs/*, bun-plugin-svelte
│   ├── bun.lock                     — Bun lockfile
│   ├── tailwind.config.js           — darkMode: 'class', skeleton wintry preset
│   └── render.yaml                  — Render static site deploy config
├── render.yaml                      — Render deploy config for service
├── wallets.config                   — comma-separated tracked wallet addresses
├── output/                          — scanner reports (gitignored)
└── AGENTS.md
```

## Architecture

1. **Scanner** — fetches leaderboard wallets from Polymarket API (`/v1/leaderboard`), enriches with closed position stats (`/closed-positions` via 5-goroutine pool with 429 retry), scores by `(PnL / TotalVolume) * 100000`, sorts descending, assigns ranks, caches result as JSON (TTL: 60m), generates static HTML report with sortable tables.

2. **Service** — loads tracked wallets from `wallets.config`, polls their open/closed positions every N minutes via Polymarket data API raw HTTP. Compares against persisted store (JSON file with `Seen`, `Open`, `Copied` maps). Detects new positions → calculates copy amount as `min(bankroll * COPY_PERCENT / 100, bankroll * MAX_OPEN_PERCENT / 100 - openCost)`, builds copy trade as `amountUSD / curPrice` → either places real CLOB order (via Go SDK with signer + API key auth) or executes paper buy (deducts from virtual balance). Skips if bankroll depleted or exposure cap hit. Detects closed positions → places sell order or paper sell (calculates PnL via close price from API).

3. **Frontend** (Svelte 5, Bun-built) — polls `/uptime`, `/status`, `/balance` every 30s, displays key metrics in cards (uptime, live/dry-run, tracked wallets, trades copied, P&L, win rate, position value, total trades), open positions table, scan status table. Dark theme via Skeleton UI v3 wintry preset.

## Config

### Service env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `COPY_PERCENT` | `2` | % of bankroll per copy trade |
| `MAX_OPEN_PERCENT` | `30` | Max % of bankroll in all open copies |
| `COPY_AMOUNT_USD` | `10` | Fallback USD per trade (real mode only) |
| `SCAN_INTERVAL_MIN` | `5` | Wallet scan interval |
| `PAPER_STAKE_USD` | `100` | Virtual stake for paper trading |
| `POLY_PRIVATE_KEY` | — | Private key for live trading |
| `POLY_API_KEY` | — | Polymarket API key |
| `POLY_API_SECRET` | — | Polymarket API secret |
| `POLY_API_PASSPHRASE` | — | Polymarket API passphrase |
| `USER_ADDRESS` | — | Your wallet address (for /balance) |
| `WALLETS_CONFIG` | `wallets.config` | File with comma-separated wallet addresses |

If `POLY_PRIVATE_KEY` + API credentials are set, live trading is enabled. Otherwise it runs in **dry-run/paper mode**.

### Scanner env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `MAX_WALLETS` | `1000` | Max wallets to track |
| `MIN_VOLUME` | `1000` | Minimum volume filter |
| `OUTPUT_PATH` | `output/report.html` | HTML report output path |
| `MARKET_LIMIT` | `20` | Max markets to fetch |
| `CACHE_PATH` | `output/.cache.json` | Cache file path |
| `CACHE_TTL_MINUTES` | `60` | Cache TTL in minutes |

### Frontend (build-time)

API URL is hardcoded in `frontend/src/api.js:1` — change `BASE` to point at your service instance.

## Deploy

### Service (Render Web Service)

- Root `render.yaml` deploys the service
- Build command: `go build -o server ./cmd/server`
- Start command: `./server`
- Set env vars via Render dashboard

### Frontend (Render Static Site)

- `frontend/render.yaml` deploys the frontend
- Build command: `bun install && bun run build`
- Publish directory: `./dist`
- Root directory: `frontend/`
- No env vars needed

## SDK Usage

Uses `github.com/GoPolymarket/polymarket-go-sdk/v2`. Read-only operations (scanner) need no auth. Order placement (service) requires a signer + API key credentials (chain ID 137 = Polygon).

## Known Issues & Fixes

- **Skeleton CSS `:root [data-theme=wintry]` selector** uses a descendant combinator (space), which doesn't match `<html data-theme="wintry">` since `<html>` IS `:root`. Fixed in `build.js` by post-processing the generated CSS: replaces `:root [data-theme=wintry]` with `[data-theme=wintry]`.
- **Service `fetchPositions` capped at 50 results** — same API pagination issue as scanner. Fixed in `fetchAllPositions` with offset-based pagination (50 per page, up to 20 pages).
- **Service 429 rate limits** — `fetchPositionsPage` now retries with exponential backoff (up to 5 attempts), same as scanner.
- **Paper/real sells used buy price** — `handleClosedTrade` was passing `info.Price` (buy price) as the sell price, making PnL always zero and real sell orders unfillable. Fixed by building a `closePriceMap` from the closed positions API response and passing the actual close/curPrice to both `Sell()` and `CloseOrder()`.

## Git History (Recent)

```
9f8c4c4 feat: percentage-based position sizing with 30% exposure cap
3f0c37a fix: paginate open/closed positions API with retry on 429, pass close price to paper/real sell orders
33a1889 fix: paginate closed-positions API to get real trade counts instead of being capped at 50
87b3a1e fix: patch :root [data-theme=wintry] -> [data-theme=wintry] so CSS vars apply on <html>
d6dfc6f ok
85896bb add bun-plugin-svelte for proper Svelte bundling
dbf37d0 remove unused Vite config files
b478824 replace Vite with Bun bundler, remove vite deps, hardcode API URL, add build.js
a699539 stack: Skeleton dark theme w/ data-theme, Svelte 5 runes, darkMode class
f47ba35 fix: import mount from svelte/client, revert base path
5737a06 fix frontend: VITE_API_URL, base path, Svelte 5 runes, loading fallback
```

## Conventions

- **No comments in code** unless necessary for a non-obvious workaround
- Use `internal/` packages for private code not meant for external import
- Errors wrapped with `fmt.Errorf("context: %w", err)`
- `context.Context` as first arg in all public functions
- HTML templates embedded via `embed.FS`
- Keep SDK abstraction in `pkg/polymarket/` — don't leak SDK types into other layers
