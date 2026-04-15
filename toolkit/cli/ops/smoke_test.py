"""
Container-Native Smoke Test Engine

CSV-driven test runner that replaces bash + jq with Python subprocess + json.
Each CSV row is executed sequentially: run szcli command, extract value by
jq_path, compare against expected_value.

Usage:
    python smoke_test.py [--flow FILE ...] [--max-retries N] [--interval S]
"""

import csv
import json
import subprocess
import sys
import time
import argparse
from pathlib import Path

DEFAULT_FLOWS = [
    "test_cases/cache_invalidation.csv",
    "test_cases/smoke-test-phase-2.csv",
    "test_cases/smoke-test-phase-3.csv",
]
DEFAULT_MAX_RETRIES = 15
DEFAULT_INTERVAL = 1


def resolve_jq_path(data, jq_path):
    """Navigate a nested dict using a jq-style dot path (e.g. '.response.data.aggregated_cases')."""
    keys = [k for k in jq_path.split(".") if k]
    current = data
    for key in keys:
        if not isinstance(current, dict):
            return None
        current = current.get(key)
        if current is None:
            return None
    return current


def coerce_and_compare(actual, expected_str):
    """Compare actual value against expected string from CSV with type coercion."""
    if actual is None:
        return str(actual) == expected_str

    # bool (JSON true/false)
    if isinstance(actual, bool):
        return str(actual).lower() == expected_str.lower()

    # int
    if isinstance(actual, int):
        try:
            return actual == int(expected_str)
        except ValueError:
            return str(actual) == expected_str

    # float
    if isinstance(actual, float):
        try:
            return abs(actual - float(expected_str)) < 0.001
        except ValueError:
            return str(actual) == expected_str

    # string
    return str(actual) == expected_str


def run_command(command, max_retries, interval):
    """Execute szcli command with smart polling. Returns parsed JSON output."""
    for attempt in range(1, max_retries + 1):
        result = subprocess.run(
            command, shell=True, capture_output=True, text=True
        )
        if result.returncode == 0 and result.stdout.strip():
            try:
                return json.loads(result.stdout)
            except json.JSONDecodeError:
                pass

        if attempt < max_retries:
            time.sleep(interval)

    print(f"  TIMEOUT after {max_retries} retries: {command}", file=sys.stderr)
    print(f"  stdout: {result.stdout}", file=sys.stderr)
    print(f"  stderr: {result.stderr}", file=sys.stderr)
    return None


def run_flow(csv_path, max_retries, interval):
    """Execute all test cases from a CSV file sequentially."""
    flow_name = Path(csv_path).stem
    print(f"\n--- Running: {flow_name} ---")

    with open(csv_path, newline="", encoding="utf-8") as f:
        reader = csv.DictReader(f)
        rows = list(reader)

    total = len(rows)
    for i, row in enumerate(rows, 1):
        command = row["szcli_command"].strip().strip('"')
        jq_path = row["jq_path"].strip().strip('"')
        expected = row["expected_value"].strip().strip('"')

        print(f"  [{i}/{total}] {command}")

        # Built-in directives
        if command.startswith("sleep "):
            seconds = float(command.split()[1])
            print(f"  WAIT — sleeping {seconds}s")
            time.sleep(seconds)
            continue

        # Retry loop: command may succeed but return stale data
        # (e.g. cache not yet invalidated), so retry assertion too.
        for attempt in range(1, max_retries + 1):
            output = run_command(command, 1, interval)
            if output is None:
                if attempt < max_retries:
                    time.sleep(interval)
                    continue
                print(f"  FAILED — command returned no output", file=sys.stderr)
                return False

            actual = resolve_jq_path(output, jq_path)
            if coerce_and_compare(actual, expected):
                break
            if attempt < max_retries:
                time.sleep(interval)
            else:
                print(f"  FAILED — {jq_path}: expected '{expected}', got '{actual}'", file=sys.stderr)
                return False

        print(f"  PASSED — {jq_path} = {actual}")

    print(f"--- {flow_name}: all {total} steps passed ---")
    return True


def cleanup():
    """Teardown: clean smoke test data from DB."""
    print("\n--- Cleanup: clearing test data ---")
    result = subprocess.run(
        "szcli -o json db clear --yes", shell=True, capture_output=True, text=True
    )
    if result.returncode == 0:
        print("  Cleanup succeeded.")
    else:
        print(f"  Cleanup failed: {result.stderr}", file=sys.stderr)


def main():
    parser = argparse.ArgumentParser(description="CSV-driven smoke test engine")
    parser.add_argument(
        "--flow", nargs="+", default=DEFAULT_FLOWS,
        help="CSV test case files to run (in order)",
    )
    parser.add_argument(
        "--max-retries", type=int, default=DEFAULT_MAX_RETRIES,
        help="Max polling retries per command",
    )
    parser.add_argument(
        "--interval", type=float, default=DEFAULT_INTERVAL,
        help="Seconds between retries",
    )
    args = parser.parse_args()

    print("=" * 60)
    print("SafeZone Container-Native Smoke Test")
    print("=" * 60)

    try:
        for csv_path in args.flow:
            if not Path(csv_path).exists():
                print(f"ERROR: {csv_path} not found", file=sys.stderr)
                sys.exit(1)
            if not run_flow(csv_path, args.max_retries, args.interval):
                sys.exit(1)

        print("\n" + "=" * 60)
        print("Smoke Test Completed Successfully")
        print("=" * 60)
    finally:
        cleanup()


if __name__ == "__main__":
    main()
