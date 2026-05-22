# Privacy And Telemetry

`pk3s` can persist a local telemetry preference, apply per-command overrides, and propagate anonymous telemetry context into delegated Productive K3S Infra or Core runs.

It can also persist an optional paid Observability bearer token used to request private event association on the telemetry backend.

## Goals

- keep telemetry explicit and auditable
- keep the shared payload contract anonymous
- let the CLI correlate a full command chain without leaking profile contents, IPs, or hostnames

## Resolution order

The CLI resolves telemetry in this order:

1. `--telemetry enable|disable` on the current command
2. `TELEMETRY_ENABLED` from the environment
3. the persisted CLI config under `~/.config/productive-k3s-cli/config.json`
4. an interactive prompt, defaulting to `Yes`
5. `false` for non-interactive runs with no prior preference

## Persisted preference

The CLI exposes explicit configuration commands:

```bash
pk3s config telemetry enable
pk3s config telemetry disable
pk3s config telemetry status
pk3s config observability set pk3s_live_xxxxx
pk3s config observability clear
pk3s config observability status
```

Those commands affect future runs. A command-level `--telemetry ...` override still wins for the current invocation.

The Observability token is independent from the telemetry consent toggle:

- if telemetry is disabled, nothing is sent even if a token exists
- if telemetry is enabled and no token exists, events remain anonymous
- if telemetry is enabled and a token exists, the CLI still sends the public marker plus `Authorization: Bearer ...`

## Propagation contract

When telemetry is enabled, the CLI sends its own events and propagates context into the delegated bundle:

- `TELEMETRY_ENABLED`
- `TELEMETRY_ENDPOINT`
- `TELEMETRY_MARKER`
- `TELEMETRY_BEARER_TOKEN`
- `TELEMETRY_MAX_RETRIES`
- `TELEMETRY_CONNECT_TIMEOUT_SECONDS`
- `TELEMETRY_REQUEST_TIMEOUT_SECONDS`
- `TELEMETRY_OUTBOX_DIR`
- `TELEMETRY_USER_AGENT`
- `TELEMETRY_SESSION_ID`
- `TELEMETRY_PARENT_RUN_ID`
- `TELEMETRY_COMPONENT`

The delegated layer then generates its own `run_id` and keeps the shared correlation chain.

Default endpoint: `https://telemetry.productive-k3s.io/telemetry`
Default marker header: `X-Productive-K3S-Telemetry: pk3s-public-v1`
Optional private header: `Authorization: Bearer <telemetry-token>`

## Correlation model

- `session_id`: shared across the whole logical operation
- `run_id`: unique to the current component execution
- `parent_run_id`: the direct parent execution in the chain

That makes it possible to reconstruct flows such as `pk3s apply -> infra scenario -> core bootstrap`.

## Privacy contract

The CLI telemetry payload is meant to include:

- command name
- target component (`infra` or `core`)
- local platform and architecture
- whether a profile-based flow was used
- anonymous correlation metadata

It is not meant to include:

- profile file contents
- IP addresses
- hostnames
- usernames
- local working directories

Telemetry remains best-effort and must never break CLI execution.
