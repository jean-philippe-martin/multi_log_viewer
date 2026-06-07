#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"

# Make sure loggen is built
go build -o bin/loggen ./cmd/loggen

# Make needed folders
mkdir -p logs/acme logs/beta
touch logs/api.log logs/acme/log.txt logs/beta/log.txt

# Execute
if [[ -x ./bin/loggen ]]; then
  LOGGEN="${LOGGEN:-./bin/loggen}"
else
  LOGGEN="${LOGGEN:-go run ./cmd/loggen}"
fi

$LOGGEN logs/api.log --interval 0.5 --timestamp-format iso8601 &
p1=$!
$LOGGEN logs/acme/log.txt --interval 0.7 --prefix "[acme] " --timestamp-format "YYYY-MM-DD HH:mm:ss.SSS" &
p2=$!
$LOGGEN logs/beta/log.txt --interval 0.9 --prefix "[beta] " --timestamp-format "YYYY-MM-DD HH:mm:ss.SSS" &
p3=$!

echo "log generators started (PIDs: $p1 $p2 $p3); Ctrl-C to stop"
trap 'kill $p1 $p2 $p3 2>/dev/null; exit 0' INT TERM
wait $p1 $p2 $p3
