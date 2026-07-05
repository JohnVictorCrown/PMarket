# PMarket — Polymarket Copy Trading Platform

Go-based system that scans top Polymarket wallets, copies their trades, and shows results in a dashboard.

## Components

| Component | Path | Description |
|-----------|------|-------------|
| **Scanner** | `scanner/` | Fetches Polymarket leaderboard & market data, scores wallets by PnL/Volume, generates HTML report |
| **Service** | `service/` | Copy-trading backend — polls tracked wallets for new/closed positions, places real or paper trades, exposes HTTP API |
| **Frontend** | `frontend/` | Svelte 5 dashboard consuming the service API — shows P&L, open positions, scan metrics |

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
npm run dev      # dev server
npm run build    # production build
```

Set `VITE_API_URL` env var to point at the service (default `http://localhost:8080`).

## Commands

- `go run ./cmd/scanner` — run scanner and generate report
- `go run ./cmd/server` — start copy-trading service
- `go test ./...` — run all tests
- `go vet ./...` — static analysis
- `golangci-lint run ./...` — full lint

## Project Structure

```
├── scanner/
│   ├── cmd/scanner/main.go
│   └── pkg/
│       ├── config/        — env-based config for scanner
│       ├── polymarket/    — read-only SDK wrapper (leaderboard, markets, closed positions)
│       ├── scanner/       — orchestrates scan, scores & ranks wallets
│       ├── report/        — HTML report generation (embed.FS templates)
│       └── types/         — Wallet, MarketSummary, ScanResult
├── service/
│   ├── cmd/server/main.go
│   └── pkg/
│       ├── config/        — env + wallets.config loading
│       ├── polymarket/    — authenticated SDK wrapper (orders, positions)
│       ├── tracker/       — periodic wallet scanner, detects new/closed trades
│       ├── store/         — persisted state (seen trades, open, copied)
│       ├── paper/         — paper trading with virtual balance
│       └── server/        — HTTP API: /health, /uptime, /status, /balance
├── frontend/
│   ├── src/
│   │   ├── App.svelte     — dashboard UI
│   │   ├── api.js         — API client
│   │   └── main.js        — Svelte entry
│   ├── index.html
│   └── vite.config.js
├── render.yaml            — Render.com deploy config for service
├── output/                — scanner reports (gitignored)
└── AGENTS.md
```

## Architecture

1. **Scanner** — fetches leaderboard wallets from Polymarket, enriches with trade stats, scores by `PnL/Volume`, generates an HTML report with sortable tables.
2. **Service** — loads tracked wallets from config, polls their open/closed positions every N minutes. Detects new positions → copies the trade (real order or paper). Detects closed positions → sells (real or paper).
3. **Frontend** — polls `/status` and `/balance` every 30s, displays key metrics, open positions, and scan status.

## Config

### Service env vars

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | HTTP port |
| `COPY_AMOUNT_USD` | `10` | USD per copied trade |
| `SCAN_INTERVAL_MIN` | `5` | Wallet scan interval |
| `PAPER_STAKE_USD` | `100` | Virtual stake for paper trading |
| `POLY_PRIVATE_KEY` | — | Private key for live trading |
| `POLY_API_KEY` | — | Polymarket API key |
| `POLY_API_SECRET` | — | Polymarket API secret |
| `POLY_API_PASSPHRASE` | — | Polymarket API passphrase |
| `USER_ADDRESS` | — | Your wallet address (for /balance) |
| `WALLETS_CONFIG` | `wallets.config` | File with comma-separated wallet addresses |

If `POLY_PRIVATE_KEY` + API credentials are set, live trading is enabled. Otherwise it runs in **dry-run/paper mode**.

## SDK Usage

Uses `github.com/GoPolymarket/polymarket-go-sdk/v2`. Read-only operations (scanner) need no auth. Order placement (service) requires a signer + API key credentials.

## Conventions

- **No comments in code** unless necessary for a non-obvious workaround
- Use `internal/` packages for private code not meant for external import
- Errors wrapped with `fmt.Errorf("context: %w", err)`
- `context.Context` as first arg in all public functions
- HTML templates embedded via `embed.FS`
- Keep SDK abstraction in `pkg/polymarket/` — don't leak SDK types into other layers
