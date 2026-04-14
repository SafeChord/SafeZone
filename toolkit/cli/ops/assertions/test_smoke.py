"""
Container-Native Smoke Test — Phase 1: Cache Invalidation Flow

Ported from scripts/smoke-test.sh test_cache_invalidation_flow().
Replaces host-coupled `docker compose logs | grep` with protocol-level
X-Cache-Status header assertions.
"""

import time

import pytest

TEST_DATE = "1970-01-01"
SIMULATE_SETTLE_SECONDS = 5


class TestCacheInvalidationFlow:
    """Verify the full cache lifecycle: MISS → HIT → invalidate → MISS → HIT."""

    def test_initial_simulate(self, dataflow_client):
        """Step 1/6: Simulate initial data."""
        resp = dataflow_client.simulate(date=TEST_DATE)
        assert resp["success"] is True

    def test_first_verify_cache_miss(self, dataflow_client):
        """Step 2/6: First verify after simulate should be a cache MISS."""
        time.sleep(SIMULATE_SETTLE_SECONDS)
        resp = dataflow_client.verify(date=TEST_DATE)
        assert resp["success"] is True
        assert resp["headers"].get("X-Cache-Status") == "MISS"

    def test_second_verify_cache_hit(self, dataflow_client):
        """Step 3/6: Second verify (same params) should be a cache HIT."""
        resp = dataflow_client.verify(date=TEST_DATE)
        assert resp["success"] is True
        assert resp["headers"].get("X-Cache-Status") == "HIT"

    def test_re_simulate_invalidates(self, dataflow_client):
        """Step 4/6: Re-simulate to trigger cache invalidation."""
        resp = dataflow_client.simulate(date=TEST_DATE)
        assert resp["success"] is True

    def test_verify_after_invalidation_cache_miss(self, dataflow_client):
        """Step 5/6: Verify after re-simulate should be a new cache MISS."""
        time.sleep(SIMULATE_SETTLE_SECONDS)
        resp = dataflow_client.verify(date=TEST_DATE)
        assert resp["success"] is True
        assert resp["headers"].get("X-Cache-Status") == "MISS"

    def test_verify_after_invalidation_cache_hit(self, dataflow_client):
        """Step 6/6: Verify again should be a new cache HIT."""
        resp = dataflow_client.verify(date=TEST_DATE)
        assert resp["success"] is True
        assert resp["headers"].get("X-Cache-Status") == "HIT"
