# Multi-log viewer

Service-centric TUI for tailing multiple log files, with health probes and start/stop scripts. Inspired by Tilt-style resource lists and multilog merge-by-timestamp.

## Requirements

- Go development environment
- Terminal with Unicode support

## Build

From the project root:

```bash
go build -o bin/mlv ./cmd/mlv
go build -o bin/loggen ./cmd/loggen
```

## Run

```
./mlv.sh
```

## Install


(then `mlv` works from any directory if `mlv.yaml` is in that directory):

```bash
cd /path/to/multi_log_viewer_go
go install ./cmd/mlv ./cmd/loggen
```

Ensure `$(go env GOPATH)/bin` is on your PATH (often `~/go/bin`):

```bash
export PATH="$(go env GOPATH)/bin:$PATH"
mlv
```

Custom config path:

```bash
./bin/mlv --config /path/to/mlv.yaml
```

## Quick start

1. Create logs (example layout):

```bash
mkdir -p logs/acme logs/beta
touch logs/api.log logs/acme/log.txt logs/beta/log.txt
```

2. Generate fake logs (optional; in a second terminal):

```bash
./run-log-generators.sh
# or manually:
./bin/loggen logs/api.log --interval 0.5 --timestamp-format iso8601
./bin/loggen logs/acme/log.txt --interval 0.7 --prefix "[acme] "
./bin/loggen logs/beta/log.txt --interval 0.9 --prefix "[beta] "
```

3. Start the viewer from the project directory (where `mlv.yaml` is):

```bash
./mlv.sh
```

## Configuration

Default config file: `mlv.yaml` in the current working directory (not the project root).

```yaml
services:
  api:
    logs:
      - path: logs/api.log
    health:
      port: 8080
    start: ./scripts/start-api.sh
    stop: ./scripts/stop-api.sh

  workers:
    logs:
      - path_pattern: "logs/{client}/log.txt"
        label: "{client}"
    health:
      process_contains: "worker.py"
```

- **path_pattern** — `{name}` captures one path segment (e.g. subfolder name as `client`).
- **health** — `process_contains` and/or `port` (all set probes must pass). Sidebar: **●** up, **○** down, **-** no probes.
- **start** / **stop** — shell command or script path.

## Keys

| Key | Action |
|-----|--------|
| j/k, ↑/↓ | Move sidebar cursor |
| Enter | Show selected log or service only (one `>` on the right) |
| + | Also show selected log or service |
| - | Remove selected log or service from main pane |
| ? | Help |
| q | Quit |

See [docs/01_initial_user_interface.md](docs/01_initial_user_interface.md) for the sidebar layout (health circles, activity bars, open-log markers).

## loggen

Test utility that appends timestamped lines to a log file:

```bash
./bin/loggen PATH [--interval SECONDS] [--prefix TEXT] [--count N] [--timestamp-format NAME]
# or: go run ./cmd/loggen PATH ...
```

`--timestamp-format` (default `hms-ms`): `hms-ms`, `hms`, `iso8601`, `iso8601-t`, `utc-hms-ms`, or aliases `YYYY-MM-DD HH:mm:ss.SSS`, `YYYY-MM-DD HH:mm:ss`, `TIMESTAMP_ISO8601`. Formats without a timezone suffix are written as UTC (matching `line_pattern` parsers in `mlv.yaml`).

## Layout

- **core** — config, patterns, tail, filters, health, engine, sidebar tree (reusable for a future web UI)
- **tui** — text-based frontend
- **loggen** — fake log writer for development

## Development

See docs/development.md