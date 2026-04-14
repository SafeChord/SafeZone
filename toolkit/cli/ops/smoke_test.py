"""
Container-Native Smoke Test

Ported from scripts/smoke-test.sh. Replaces host-coupled bash + jq + docker logs
with Python client calls and X-Cache-Status header assertions.

Usage:
    python smoke_test.py
"""

import sys
import time

from bin.client import DataflowClient

SIMULATE_SETTLE_SECONDS = 5


# --- Logging helpers ---

def log(phase, step, msg):
    print(f"[{phase}] Step {step}: {msg}")


def log_pass(phase, step, msg):
    print(f"[{phase}] Step {step}: PASSED — {msg}")


def fail(phase, step, msg):
    print(f"[{phase}] Step {step}: FAILED — {msg}", file=sys.stderr)
    sys.exit(1)


def verify_and_check(client, phase, step, msg, expected_field, expected_value, **kwargs):
    """Call client.verify() and assert a specific field value."""
    log(phase, step, msg)
    resp = client.verify(**kwargs)
    if not resp["success"]:
        fail(phase, step, f"Verify failed: {resp.get('message')}")

    # Navigate nested fields (e.g. "data.aggregated_cases")
    actual = resp
    for key in expected_field.split("."):
        actual = actual.get(key) if isinstance(actual, dict) else None
        if actual is None:
            fail(phase, step, f"Field '{expected_field}' not found in response")

    # Compare with type coercion for numeric values
    if isinstance(expected_value, float):
        if abs(float(actual) - expected_value) > 0.001:
            fail(phase, step, f"Expected {expected_field}={expected_value}, got {actual}")
    elif actual != expected_value:
        fail(phase, step, f"Expected {expected_field}={expected_value}, got {actual}")

    log_pass(phase, step, f"{expected_field}={actual}")


# --- Phase 1: Cache Invalidation ---

def phase1_cache_invalidation(client: DataflowClient):
    """Cache invalidation lifecycle: MISS -> HIT -> invalidate -> MISS -> HIT."""
    test_date = "1970-01-01"

    # Step 1: Simulate initial data
    log("P1", "1/6", f"Simulating initial data for {test_date}...")
    resp = client.simulate(date=test_date)
    if not resp["success"]:
        fail("P1", "1/6", f"Simulate failed: {resp.get('message')}")
    log_pass("P1", "1/6", "Simulate succeeded.")
    time.sleep(SIMULATE_SETTLE_SECONDS)

    # Step 2: First verify — expect MISS
    log("P1", "2/6", "Verifying data (expecting CACHE MISS)...")
    resp = client.verify(date=test_date)
    if not resp["success"]:
        fail("P1", "2/6", f"Verify failed: {resp.get('message')}")
    cache_status = resp["headers"].get("X-Cache-Status", "")
    if cache_status != "MISS":
        fail("P1", "2/6", f"Expected MISS, got '{cache_status}'")
    log_pass("P1", "2/6", "Cache MISS confirmed.")

    # Step 3: Second verify — expect HIT
    log("P1", "3/6", "Verifying again (expecting CACHE HIT)...")
    resp = client.verify(date=test_date)
    cache_status = resp["headers"].get("X-Cache-Status", "")
    if cache_status != "HIT":
        fail("P1", "3/6", f"Expected HIT, got '{cache_status}'")
    log_pass("P1", "3/6", "Cache HIT confirmed.")

    # Step 4: Re-simulate to trigger invalidation
    log("P1", "4/6", "Re-simulating to trigger cache invalidation...")
    resp = client.simulate(date=test_date)
    if not resp["success"]:
        fail("P1", "4/6", f"Re-simulate failed: {resp.get('message')}")
    log_pass("P1", "4/6", "Re-simulate succeeded.")
    time.sleep(SIMULATE_SETTLE_SECONDS)

    # Step 5: Verify after invalidation — expect MISS
    log("P1", "5/6", "Verifying after invalidation (expecting CACHE MISS)...")
    resp = client.verify(date=test_date)
    cache_status = resp["headers"].get("X-Cache-Status", "")
    if cache_status != "MISS":
        fail("P1", "5/6", f"Expected MISS, got '{cache_status}'")
    log_pass("P1", "5/6", "New Cache MISS confirmed after invalidation.")

    # Step 6: Verify again — expect HIT
    log("P1", "6/6", "Verifying again (expecting CACHE HIT)...")
    resp = client.verify(date=test_date)
    cache_status = resp["headers"].get("X-Cache-Status", "")
    if cache_status != "HIT":
        fail("P1", "6/6", f"Expected HIT, got '{cache_status}'")
    log_pass("P1", "6/6", "New Cache HIT confirmed after invalidation.")


# --- Phase 2: Single-day data verification ---

def phase2_single_day(client: DataflowClient):
    """Verify single-day queries against known test data (1970-01-01)."""

    # Seed: simulate single day
    log("P2", "1/6", "Simulating 1970-01-01...")
    resp = client.simulate(date="1970-01-01")
    if not resp["success"]:
        fail("P2", "1/6", f"Simulate failed: {resp.get('message')}")
    log_pass("P2", "1/6", "Simulate succeeded.")
    time.sleep(SIMULATE_SETTLE_SECONDS)

    # National
    verify_and_check(client, "P2", "2/6",
        "Verify national aggregated_cases",
        "data.aggregated_cases", 7,
        date="1970-01-01")

    # City: 台北市
    verify_and_check(client, "P2", "3/6",
        "Verify city=台北市 aggregated_cases",
        "data.aggregated_cases", 7,
        date="1970-01-01", city="台北市")

    # Region: 台北市/中山區
    verify_and_check(client, "P2", "4/6",
        "Verify city=台北市, region=中山區 aggregated_cases",
        "data.aggregated_cases", 2,
        date="1970-01-01", city="台北市", region="中山區")

    # Region with ratio
    verify_and_check(client, "P2", "5/6",
        "Verify city=台北市, region=中山區 ratio",
        "data.cases_population_ratio", 0.09244,
        date="1970-01-01", city="台北市", region="中山區", ratio=True)

    # City with zero cases: 台中市
    verify_and_check(client, "P2", "6/6",
        "Verify city=台中市 aggregated_cases (expect 0)",
        "data.aggregated_cases", 0,
        date="1970-01-01", city="台中市")


# --- Phase 3: Multi-day interval verification ---

def phase3_interval(client: DataflowClient):
    """Verify interval queries against known test data (1970-01-01 ~ 1970-01-30)."""

    # Seed: simulate 30-day range
    log("P3", "1/12", "Simulating 1970-01-01 ~ 1970-01-30...")
    resp = client.simulate(date="1970-01-01", end_date="1970-01-30")
    if not resp["success"]:
        fail("P3", "1/12", f"Simulate failed: {resp.get('message')}")
    log_pass("P3", "1/12", "Simulate succeeded.")
    time.sleep(SIMULATE_SETTLE_SECONDS)

    # 30-day national
    verify_and_check(client, "P3", "2/12",
        "Verify 30-day national aggregated_cases",
        "data.aggregated_cases", 2228,
        date="1970-01-30", interval=30)

    # 30-day city with zero: 台中市
    verify_and_check(client, "P3", "3/12",
        "Verify 30-day city=台中市 (expect 0)",
        "data.aggregated_cases", 0,
        date="1970-01-30", interval=30, city="台中市")

    # 30-day region: 台北市/中山區
    verify_and_check(client, "P3", "4/12",
        "Verify 30-day city=台北市, region=中山區",
        "data.aggregated_cases", 1004,
        date="1970-01-30", interval=30, city="台北市", region="中山區")

    # 14-day region from end (1970-01-30): 台北市/中山區
    verify_and_check(client, "P3", "5/12",
        "Verify 14-day (from 01-30) city=台北市, region=中山區",
        "data.aggregated_cases", 1,
        date="1970-01-30", interval=14, city="台北市", region="中山區")

    # 14-day region from mid (1970-01-14): 台北市/中山區
    verify_and_check(client, "P3", "6/12",
        "Verify 14-day (from 01-14) city=台北市, region=中山區",
        "data.aggregated_cases", 1003,
        date="1970-01-14", interval=14, city="台北市", region="中山區")

    # 14-day region with ratio
    verify_and_check(client, "P3", "7/12",
        "Verify 14-day city=台北市, region=中山區 ratio",
        "data.cases_population_ratio", 46.36092,
        date="1970-01-14", interval=14, city="台北市", region="中山區", ratio=True)

    # 7-day national
    verify_and_check(client, "P3", "8/12",
        "Verify 7-day national aggregated_cases",
        "data.aggregated_cases", 17,
        date="1970-01-07", interval=7)

    # 3-day national
    verify_and_check(client, "P3", "9/12",
        "Verify 3-day national aggregated_cases",
        "data.aggregated_cases", 6,
        date="1970-01-07", interval=3)

    # Single-day region (re-verify after interval seed)
    verify_and_check(client, "P3", "10/12",
        "Verify single-day city=台北市, region=中山區",
        "data.aggregated_cases", 2,
        date="1970-01-01", city="台北市", region="中山區")

    # Single-day region with ratio (re-verify)
    verify_and_check(client, "P3", "11/12",
        "Verify single-day city=台北市, region=中山區 ratio",
        "data.cases_population_ratio", 0.09244,
        date="1970-01-01", city="台北市", region="中山區", ratio=True)

    log_pass("P3", "12/12", "All interval verifications passed.")


# --- Main ---

def main():
    print("=" * 60)
    print("SafeZone Container-Native Smoke Test")
    print("=" * 60)

    dataflow = DataflowClient()

    print("\n--- Phase 1: Cache Invalidation Mechanism ---")
    phase1_cache_invalidation(dataflow)

    print("\n--- Phase 2: Single-Day Data Verification ---")
    phase2_single_day(dataflow)

    print("\n--- Phase 3: Multi-Day Interval Verification ---")
    phase3_interval(dataflow)

    print("\n" + "=" * 60)
    print("Smoke Test Completed Successfully")
    print("=" * 60)


if __name__ == "__main__":
    main()
