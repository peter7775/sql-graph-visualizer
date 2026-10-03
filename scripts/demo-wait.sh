#!/bin/sh
# Waits until the live-demo containers report "healthy" (used by `make demo`).
#
# Usage: scripts/demo-wait.sh [container ...]
#   default containers: mysql-demo neo4j-demo sql-graph-demo
# Environment: DEMO_WAIT_TIMEOUT (seconds per container, default 600)
#
# The first start is slow because MySQL seeds ~800k rows and Neo4j initializes its store;
# progress is printed whenever a container changes state.
set -u

TIMEOUT="${DEMO_WAIT_TIMEOUT:-600}"
CONTAINERS="${*:-mysql-demo neo4j-demo sql-graph-demo}"

state_of() {
  docker inspect -f '{{if .State.Health}}{{.State.Health.Status}}{{else}}{{.State.Status}}{{end}}' "$1" 2>/dev/null || echo "missing"
}

describe() {
  case "$1" in
    mysql-demo) echo "MySQL (first start seeds the e-shop dataset, takes about 1-2 minutes)" ;;
    neo4j-demo) echo "Neo4j" ;;
    sql-graph-demo) echo "application" ;;
    *) echo "$1" ;;
  esac
}

for name in $CONTAINERS; do
  label=$(describe "$name")
  echo "Waiting for $label ..."
  waited=0
  last=""
  while :; do
    state=$(state_of "$name")
    if [ "$state" != "$last" ]; then
      echo "  $name: $state"
      last="$state"
    fi
    case "$state" in
      healthy) break ;;
      exited|dead|missing)
        echo "ERROR: $name is $state. Inspect with: docker logs $name" >&2
        exit 1
        ;;
    esac
    if [ "$waited" -ge "$TIMEOUT" ]; then
      echo "ERROR: timed out after ${TIMEOUT}s waiting for $name (state: $state). Inspect with: docker logs $name" >&2
      exit 1
    fi
    sleep 3
    waited=$((waited + 3))
  done
done

echo "All demo containers are healthy."
