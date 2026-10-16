#!/usr/bin/env python3
"""Wait for the worker's consume-loop health endpoint during the stack smoke test."""

import sys
import time
import urllib.error
import urllib.request


HEALTH_URL = "http://worker-golang:8080/health"
MAX_ATTEMPTS = 15


def main() -> int:
    last_error = "no response"
    for attempt in range(1, MAX_ATTEMPTS + 1):
        try:
            with urllib.request.urlopen(HEALTH_URL, timeout=2) as response:
                if response.status == 200:
                    print(f"Worker health check passed: {HEALTH_URL}")
                    return 0
                last_error = f"HTTP {response.status}"
        except (urllib.error.URLError, TimeoutError) as exc:
            last_error = str(exc)

        if attempt < MAX_ATTEMPTS:
            time.sleep(1)

    print(
        f"Worker health check failed after {MAX_ATTEMPTS} attempts: {last_error}",
        file=sys.stderr,
    )
    return 1


if __name__ == "__main__":
    raise SystemExit(main())
