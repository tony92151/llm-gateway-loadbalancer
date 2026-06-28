# LLM Gateway Load Balancer

OpenAI-compatible LLM gateway and API-key load balancer written in Go. It exposes a local `/v1/*` proxy endpoint, rotates traffic across multiple upstream API keys, records usage and cost into SQLite, and serves lightweight admin/monitor endpoints for runtime visibility.

## Features

- OpenAI-compatible proxy for `/v1/*` upstream APIs.
- Multiple upstream API keys with `leastload`, `roundrobin`, or `weighted` selection.
- Per-key runtime state: enabled flag, in-flight requests, RPM/TPM counters, cooldown, and last error.
- Retry on upstream `429` and `5xx` responses with the next available key.
- SSE streaming pass-through for chat completion streams.
- Usage extraction from OpenAI-style `usage` payloads, including cached prompt tokens.
- Cost calculation from per-model pricing in config.
- SQLite persistence for request logs, key state, model prices, and hourly aggregates.
- Admin JSON APIs and a built-in browser monitor.

## Requirements

- Go 1.24 or newer.
- An OpenAI-compatible upstream endpoint.
- One or more upstream API keys.

## Quick Start

Create a local config from the example:

```bash
cp configs/config.yaml.example config.yaml
```

Set the API keys referenced by the example config:

```bash
export UPSTREAM_KEY_A="sk-..."
export UPSTREAM_KEY_B="sk-..."
```

Run migrations explicitly if desired:

```bash
go run ./cmd/lgl migrate ./config.yaml
```

Start the gateway:

```bash
go run ./cmd/lgl serve ./config.yaml
```

By default, the proxy listens on `http://0.0.0.0:8787` and the admin/monitor server listens on `http://127.0.0.1:8789`.

You can also build a single binary:

```bash
go build -o lgl ./cmd/lgl
./lgl serve ./config.yaml
```

## Proxy Usage

Point an OpenAI-compatible client at the gateway base URL:

```bash
curl http://127.0.0.1:8787/v1/chat/completions \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer local-client-token" \
  -d '{
    "model": "gpt-4.1-mini",
    "messages": [{"role": "user", "content": "Hello"}]
  }'
```

The gateway forwards `/v1/*` requests to `upstream.base_url`, replaces the outbound `Authorization` header with the selected upstream key, and records the request result in SQLite.

### Python OpenAI SDK Example

Install the Python SDK:

```bash
python -m pip install openai
```

Start the gateway with your local config:

```bash
go run ./cmd/lgl serve ./config.yaml
```

Then point the OpenAI client at the gateway. Match `LGL_BASE_URL` to `server.port` in `config.yaml`, and use one of the enabled model names from `upstream.models`.

```bash
export LGL_BASE_URL="http://127.0.0.1:8781/v1"
export LGL_MODEL="lightning-ai/minimax-m2.5"

python - <<'PY'
import os
from openai import OpenAI

client = OpenAI(
    base_url=os.environ["LGL_BASE_URL"],
    api_key="local-client-token",
)

print("Available models:")
for model in client.models.list().data:
    print("-", model.id)

response = client.chat.completions.create(
    model=os.environ["LGL_MODEL"],
    messages=[
        {"role": "user", "content": "Say hello in one short sentence."},
    ],
)

print(response.choices[0].message.content)
PY
```

The inbound `api_key` only needs to be a non-empty client token for SDK compatibility; the gateway replaces it with a selected upstream key before forwarding the request.

## Configuration

See [configs/config.yaml.example](configs/config.yaml.example) for a complete example.

Important sections:

- `server`: public proxy listener and request timeouts.
- `monitor`: admin/monitor listener. When `monitor.enabled` is true, `/monitor/` serves the built-in dashboard.
- `upstream`: upstream base URL, HTTP client tuning, enabled models, pricing, and API keys.
- `selector`: key selection strategy, health-check interval, cooldown, and max retries.
- `database`: SQLite path, connection limits, and WAL mode.
- `logging`: zerolog level and output format.

API key values support environment expansion in YAML, for example:

```yaml
keys:
  - label: "key-a"
    key: "${UPSTREAM_KEY_A}"
    weight: 10
    rpm_limit: 0
    tpm_limit: 0
    enabled: true
```

Supported selector strategies are:

- `leastload`: choose the available key with the fewest in-flight requests, using weight as a tie-breaker.
- `roundrobin`: cycle through available keys.
- `weighted`: randomly select among available keys according to configured weights.

`token_balance` is intentionally rejected by config validation in the current MVP.

## Admin And Monitor

Admin routes are served on the monitor/admin listener:

- `GET /admin/health`
- `GET /admin/keys`
- `GET /admin/requests/recent?limit=100`
- `GET /admin/metrics/summary?window=1h`
- `GET /admin/dashboard?window=15m|1h|24h`
- `GET /monitor/` when `monitor.enabled` is true

Open the monitor UI locally:

```bash
open http://127.0.0.1:8789/monitor/
```

The admin and monitor endpoints currently do not implement authentication. Bind them to localhost or put them behind trusted network controls before exposing them.

## Runtime Behavior

On startup, `lgl serve` creates the SQLite directory if needed, opens the database, applies migrations, initializes key state, and starts both the proxy server and the admin/monitor server.

Health checks run periodically against `${upstream.base_url}/models` using each enabled key. Successful checks clear key errors and cooldowns; failed checks mark the key unhealthy for the configured cooldown.

Request logs are recorded for completed non-streaming requests, completed streaming requests, and cases where no upstream key is available. Usage and cost are populated when the upstream response includes an OpenAI-style `usage` object.

Hourly aggregates are refreshed by a scheduler for the previous UTC hour.

## Development

Run the full test suite:

```bash
go test ./...
```

Check the CLI version:

```bash
go run ./cmd/lgl version
```

Useful source entry points:

- [cmd/lgl/main.go](cmd/lgl/main.go): CLI commands.
- [internal/app/app.go](internal/app/app.go): application wiring and server lifecycle.
- [internal/proxy/handler.go](internal/proxy/handler.go): proxy forwarding, retries, streaming, and recording.
- [internal/selector/selector.go](internal/selector/selector.go): key selection and rate-limit state.
- [internal/httpserver/admin.go](internal/httpserver/admin.go): admin API routes.
- [internal/monitor/static/index.html](internal/monitor/static/index.html): built-in monitor UI.
- [internal/store/migrate.go](internal/store/migrate.go): SQLite schema.

## Current Limits

- The gateway does not authenticate inbound clients.
- Upstream API keys are read from config/env at startup and are not encrypted into the database.
- There are no runtime key-management CLI commands yet.
- Streaming responses are proxied and logged, but token usage/cost is only captured when a non-streaming response body contains `usage`.
