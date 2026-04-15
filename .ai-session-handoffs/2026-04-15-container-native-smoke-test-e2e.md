# Session Handoff: Container-Native Smoke Test E2E Verification

**Date**: 2026-04-15
**Agent**: Claude Code (Pioneer)
**Branch**: `feat/container-native-assertions`
**PR**: https://github.com/SafeChord/SafeZone/pull/34 (→ dev)

---

## Summary

Completed the E2E verification of the container-native assertions framework (spike from 2026-04-14). The CSV-driven smoke test engine now runs inside an ops container on the compose network, replacing the old bash+jq smoke test as the default `make smoke-test` target. Local CI (`make local-ci` via `act`) passed green.

## Verified State

- **`make smoke-test`**: 27/27 test steps passed (cache_invalidation + phase-2 + phase-3)
- **`make local-ci`**: Full pipeline green (build-all → test-all → smoke-test)
- **`make dev-up` / `make dev-down`**: Working with new compose paths
- **X-Cache-Status header pipeline**: Verified end-to-end (analytics-api → relay → szcli)
- **`szcli db clear --yes`**: Non-interactive cleanup working in container context

## Key Changes in This Session

| File | Change |
|---|---|
| `scripts/smoke-test.sh` | Replaced old bash+jq script with container-native runner |
| `scripts/dev-up.sh` | Fixed `((attempt++))` bash arithmetic bug with `set -e` |
| `scripts/dev-down.sh` | Updated compose file path |
| `makefile` | `smoke-test` target points to new runner; removed old CSV path vars |
| `docker-compose/local-compose-all.yml` | Consolidated from `smoke-test/release_0.2.0.yml`, fixed `env_file` relative paths |
| `toolkit/cli/ops/smoke_test.py` | Added assertion retry loop, `sleep` directive, `--yes` on db clear |
| `toolkit/cli/ops/test_cases/*.csv` | Fixed header case (`x-cache-status`), added sleep directives |
| `toolkit/cli/command/main.py` | Added `--yes`/`-y` flag to `db clear` command |
| git remote | Updated origin from `rebodutch/SafeZone` to `SafeChord/SafeZone` |

## Bugs Found & Fixed

1. **`dev-up.sh` health check crash**: `((attempt++))` returns exit code 1 when attempt=0 under `set -e`. Fixed with `attempt=$((attempt + 1))`.
2. **CSV header case mismatch**: CSV used `X-Cache-Status` (Pascal) but `requests` library normalizes to `x-cache-status` (lowercase).
3. **`db clear` interactive prompt**: `typer.confirm()` aborts in non-TTY containers. Added `--yes` flag to bypass.
4. **Kafka consumer lag**: Simulate returns success before worker writes to DB. Added `sleep` CSV directive and assertion retry.

## Design Decisions

- **`cli-daemon` kept in compose**: Needed for `dev-up.sh` db init and interactive dev use (`docker exec`). Ops container is for automated testing only.
- **Assertion retry at flow level**: Rather than adding explicit sleep between every step, the engine retries the full command+assertion up to `max_retries` times. Sleep directives are only for known async gaps (simulate → Kafka → worker → DB).
- **Volume-mount mode deferred**: CSV and engine are baked into ops image. Mount-based iteration mode not yet implemented (0.x, forward fix).

## Next Actions (for Settler/Gemini)

1. **Review PR #34** — check for consistency with SSOT docs
2. **Docs reconciliation** — update KDD if smoke test architecture documentation exists
3. **Consider**: old `data/smoke-test/` CSV files deleted, test cases now live in `toolkit/cli/ops/test_cases/`
