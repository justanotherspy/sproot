#!/usr/bin/env bash
# Retry a command with exponential backoff. The integration jobs create real
# sprites over the network, which occasionally fails transiently (e.g. the
# sprite API returning "connection reset by peer" on the create POST). Wrapping
# those commands here lets a flake ride out instead of failing the job and
# forcing a manual workflow replay.
#
# Usage:
#   retry.sh <command> [args...]
#
# Env knobs:
#   RETRY_ATTEMPTS  total attempts before giving up (default 3)
#   RETRY_DELAY     base backoff in seconds, doubled after each failure (default 5)
#   RETRY_CLEANUP   optional shell snippet run between failed attempts (its own
#                   errors are ignored). Used to tear down a half-created sprite
#                   before retrying so the next attempt does not collide on name.
set -uo pipefail

attempts="${RETRY_ATTEMPTS:-3}"
delay="${RETRY_DELAY:-5}"

if [ "$#" -eq 0 ]; then
  echo "retry: no command given" >&2
  exit 2
fi

n=1
while true; do
  "$@"
  status=$?
  if [ "$status" -eq 0 ]; then
    exit 0
  fi
  if [ "$n" -ge "$attempts" ]; then
    echo "retry: command failed after ${n} attempt(s) (last exit ${status}): $*" >&2
    exit "$status"
  fi
  if [ -n "${RETRY_CLEANUP:-}" ]; then
    echo "retry: running cleanup before next attempt" >&2
    bash -c "${RETRY_CLEANUP}" || true
  fi
  echo "retry: attempt ${n}/${attempts} failed (exit ${status}); retrying in ${delay}s: $*" >&2
  sleep "$delay"
  n=$((n + 1))
  delay=$((delay * 2))
done
