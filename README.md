# Rune Pack

**Go** · **v0.1.0**

_URL shortener HTTP service with JSON file persistence and click tracking_

---

## Features

- POST /api/shorten generates a random 6-character code
- GET /<code> issues a 302 redirect and counts clicks
- Atomic JSON persistence survives restarts, zero external dependencies

## Installation

```bash
git clone https://github.com/DevCode01/rune-pack.git
cd rune-pack
```

## Usage

```bash
PORT=8080 go run main.go
```

## How it works

An in-memory map backed by a JSON file persisted on every write (atomic rename). `POST /api/shorten` generates a random 6-character code and stores the target URL; `GET /<code>` looks it up, increments its click counter, persists, and issues a 302 redirect; `GET /api/stats/<code>` returns the URL and click count.

## Options

- `PORT` - environment variable, listening port (default 8080)
- `SHORTENER_DB` - environment variable, JSON persistence file (default shortener.json)
- `POST /api/shorten` - body `{"url": "https://..."}`, returns `{"code", "short_url"}`
- `GET /<code>` - 302 redirect to the stored URL, counts the click
- `GET /api/stats/<code>` - returns `{"url", "clicks", "created_at"}`

## Example

```bash
$ PORT=8080 go run main.go
URL shortener listening on :8080 (db: shortener.json)
```

## Testing

```bash
bash tests/run.sh
```

Runs the build/syntax check (interactive/random/networked output isn't asserted exactly).

## License

MIT
