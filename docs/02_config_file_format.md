# Configuration file format

Multi-log viewer reads a single YAML file (default: `mlv.yaml` in the current working directory). Pass another path with:

```bash
mlv --config /path/to/mlv.yaml
```

Paths in the config are resolved relative to the **project root** (see below), not necessarily the shell’s current directory.

## Top-level fields

| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `project_root` | string (path) | directory containing the config file | Base directory for log paths, patterns, and service `cwd`. |
| `health_poll_seconds` | number | `2` | How often to re-run health probes (seconds). |
| `pattern_rescan_seconds` | number | `5` | How often to re-scan glob patterns for new or removed log files. |
| `ring_buffer_lines` | integer | `2000` | Maximum lines kept in memory per log file. |
| `theme` | string | `default` | UI color theme; loads `themes/<name>.yaml` under the project root (e.g. `default`, `vibrant`). |
| `services` | mapping | *(required)* | Named services; keys become service IDs in the UI. |

Example skeleton:

```yaml
health_poll_seconds: 2
pattern_rescan_seconds: 5
ring_buffer_lines: 2000
theme: default

services:
  api:
    # ...
```

## Services

Each entry under `services` is one row in the sidebar (e.g. `api`, `workers`). The YAML key is the **service id** (shown as the service name).

A service must define **`logs`**: a non-empty list of log entries (see below). Other fields are optional.

| Field | Type | Description |
|-------|------|-------------|
| `logs` | list | **Required.** Where to find log files for this service. |
| `health` | mapping | Optional probes for whether the service is “up”. |
| `start` | string or list | Shell command to start the service (loaded by the engine; not bound in the initial TUI). |
| `stop` | string or list | Shell command to stop the service (same note). |
| `cwd` | string (path) | Working directory for `start` / `stop`, and base for this service’s log paths if set. Resolved under `project_root`. |

### `start` and `stop`

- **String** — run with `/bin/sh -c` (same as a one-line shell script).
- **List** — argv passed to `exec` (no shell), e.g. `["python", "server.py"]`.

Commands run with `cwd` set to `project_root` or `project_root / cwd` when `cwd` is set.

## Log entries

Each item under `logs` describes one way to discover files. Every entry must have exactly one of **`path`** or **`path_pattern`** (not both).

### Fixed path (`path`)

```yaml
logs:
  - path: logs/api.log
```

- Resolved as `project_root / path` (or `project_root / cwd / path` when the service has `cwd`).
- Parent directories are created if missing (empty file can be tailed once created).
- In the sidebar: one **log** row under the service, labeled with the file name (e.g. `api.log`).

Optional **`label`** overrides the display name; for a single file it is rarely needed.

### Pattern (`path_pattern`)

```yaml
logs:
  - path_pattern: "logs/{client}/log.txt"
    label: "{client}"
```

- **`path_pattern`** — path template relative to the service base directory. Brace segments `{name}` match a single path component (no `/`). At runtime each `{name}` becomes a `*` for globbing, so `logs/{client}/log.txt` matches `logs/acme/log.txt`, `logs/beta/log.txt`, etc.
- **`label`** — optional display template. `{client}` (and other capture names) are substituted from the match. If omitted, captured values are joined with `/` (e.g. `acme` or `acme/prod`).

Capture variables are stored per file and used to **group** logs in the sidebar: all logs sharing the same captures appear under one folder row (▷), with one `- filename` row per file.

Example with multiple files per client:

```yaml
workers:
  logs:
    - path_pattern: "logs/{client}/stdout.txt"
    - path_pattern: "logs/{client}/stderr.txt"
  health:
    process_contains: "worker.py"
```

Both `path_pattern`s share `{client}`, so `acme` gets a folder with `stdout.txt` and `stderr.txt` underneath.

Rules:

- One log entry cannot include both `path` and `path_pattern`.
- Each entry must include at least one of them.
- Only regular files are tailed; directories matched by glob are skipped.

## Log lines pattern (`line_pattern`)

```yaml
logs:
  - path: logs/api.log
    line_pattern: "{timestamp:TIMESTAMP_ISO8601} [{worker}] {level:LOGLEVEL}"
```

The line_pattern is used by our program to find the timestamp and log level in the log file, and
optionally other fields to filter on (not implemented yet). JSON log lines are not supported yet.

Here is an example line that mathces the line pattern above:
```
2026-05-31 17:06:15.123 -07:00 [alpha] INFO the foojigger has froobared
```

The idea is that the string is anchored at the beginning of the line. Characters are treated as literal ("log line must match this") but what's in the brackets
is a variable name and optional `:` then a type. The only supported types at this point are:

- TIMESTAMP_ISO8601 e.g. 2026-05-31 17:06:15.123 -07:00. Timezone taken from the offset in the line.
- YYYY-MM-DD HH:mm:ss - missing a timezone info so we assume UTC.
- YYYY-MM-DD HH:mm:ss.SSS - includes milliseconds. We still assume UTC.
- LOGLEVEL: enum type ['DEBUG', 'INFO', 'WARN', 'ERROR']
- if no type is indicated, then the string type is assumed

The variables don't mean "must watch the value of this variable that's defined elsewhere", but it means:
"the string from the log line gives a value to the line variable you named here"

The actual line in the file may be longer than line_pattern, we still save and display the rest as well.

line variable "timestamp" tells the system at what time that line was written to the log.

If a line does not match the pattern then no line variable is set, but the line will still be displayed.

## Health

```yaml
health:
  port: 8080
  process_contains: "myapp.py"
```

| Field | Type | Description |
|-------|------|-------------|
| `port` | integer or string | TCP connect check. Integer → `127.0.0.1:port`. String may be `host:port` (e.g. `0.0.0.0:8080`). |
| `process_contains` | string | Substring search in process command lines (case-insensitive). |

If **both** are set, **all** configured probes must pass for the service to count as running.

If `health` is omitted, or the block is empty, the service has no liveness probes.

### Sidebar display (initial UI)

Liveness symbol on each **service** row (first column after the cursor):

| Symbol | Meaning |
|--------|---------|
| **●** | At least one health probe is configured and all pass (service is up). |
| **○** | Health probes are configured but at least one fails (service looks down). |
| **-** | No health probes (`health` omitted or empty). |

Probes include **`port`** (TCP connect) and **`process_contains`** (command-line substring). Either or both can be set; all configured probes must pass for **●**.

## Project root and paths

| Concept | Resolution |
|---------|------------|
| Default `project_root` | Parent directory of the config file |
| Override | `project_root: ..` or any relative/absolute path relative to the config file’s directory |
| Service log paths | `(project_root / service.cwd?) / path-or-pattern` |

Run `mlv` from any directory if you pass `--config`; paths still anchor on `project_root`, not on the shell cwd.

## Full example

```yaml
health_poll_seconds: 2
pattern_rescan_seconds: 5
ring_buffer_lines: 2000

services:
  api:
    logs:
      - path: logs/api.log
    health:
      port: 8080
    start: ./scripts/start-api.sh
    stop: ./scripts/stop-api.sh

  workers:
    cwd: .
    logs:
      - path_pattern: "logs/{client}/log.txt"
        label: "{client}"
    health:
      process_contains: "worker.py"
```

## Validation errors

The loader fails fast with clear errors, including:

- `'services' must be a mapping`
- `service '<id>' must define logs`
- `log entry requires path or path_pattern`
- `log entry cannot have both path and path_pattern`

Missing config file is reported by the CLI before load.

## Relation to the UI

See [01_initial_user_interface.md](01_initial_user_interface.md) for how services, folders, and files appear in the sidebar and how open logs are chosen in the TUI.

| Config | Sidebar |
|--------|---------|
| `services` key | Service row |
| Logs sharing the same `{capture}` values | Folder row (▷) + file rows |
| Single `path` log (no captures) | File row directly under service (`- filename`) |
| `health.port` | ○ / ● / - on the service row (see [Health](#health)) |

The config format is intentionally small so the core engine can stay reusable if a web or other frontend is added later.
