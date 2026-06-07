# development

## Prerequisites

- Go 1.22+
- Optional: `golangci-lint` (otherwise `run-tests.sh` runs it via `go run`)

## Build

```bash
go build -o bin/mlv ./cmd/mlv
go build -o bin/loggen ./cmd/loggen
```

Or install:

```bash
go install ./cmd/mlv ./cmd/loggen
```

## Run

From the project directory (so `./mlv.yaml` is found):

```bash
go run ./cmd/mlv
# or: mlv --config /path/to/mlv.yaml
```

## Test logs

```bash
./run-log-generators.sh
```

Starts `loggen` on the sample `logs/` layout. Stop with Ctrl-C.

Manual example:

```bash
go run ./cmd/loggen logs/api.log --interval 0.5 --timestamp-format iso8601
```

## Tests and lint

```bash
./run-tests.sh
```

Runs `go vet`, `go test ./...`, and `golangci-lint`.

Pass extra arguments to `go test`:

```bash
./run-tests.sh -v -run TestLoadValid
```
