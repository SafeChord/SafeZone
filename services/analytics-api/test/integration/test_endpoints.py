"""Integration tests for API endpoints — scaffold-compliant.

Rules:
- Uses TestClient with dependency_overrides (not request.app.state passthrough)
- Import path: from main import create_app (unchanged)
- Parametrized via test/cases/test_integration.json
- Cache decorator should gracefully skip when no Redis is available
"""
import json

import pytest


with open("/test/cases/test_integration.json", encoding="utf-8") as f:
    test_cases = json.load(f)


def get_case_describes(case):
    return case["test_describes"]


@pytest.mark.parametrize("case", test_cases, ids=get_case_describes)
def test_endpoint(case, client):
    response = client.get(case["endpoint"], params=case["params"])
    assert response.status_code == case["expected"]["status_code"]

    response_body = response.json()
    if "timestamp" in response_body:
        del response_body["timestamp"]

    if "data" in response_body:
        response_body["data"] = {
            k: v for k, v in response_body["data"].items() if v is not None
        }
    response_body = {k: v for k, v in response_body.items() if v is not None}

    assert response_body == case["expected"]["response"]


def test_cache_miss_calls_handler_and_stores(client_with_cache, mock_cache):
    """MISS: handler is called, result is stored via setex."""
    from unittest.mock import AsyncMock
    mock_cache.get = AsyncMock(return_value=None)
    mock_cache.setex.reset_mock()

    response = client_with_cache.get("/cases/national", params={"now": "2023-03-26", "interval": 7})
    assert response.status_code == 200
    mock_cache.setex.assert_called_once()


def test_cache_hit_returns_cached_skips_handler(client_with_cache, mock_cache):
    """HIT: cached value returned, setex NOT called (handler bypassed)."""
    from unittest.mock import AsyncMock
    from utils.pydantic_model.response import AnalyticsAPIResponse, AnalyticsAPIData

    sentinel = AnalyticsAPIResponse(
        success=True,
        message="Data returned successfully",
        detail="Data returned successfully for dates 2023-03-20 ~ 2023-03-26.",
        data=AnalyticsAPIData(start_date="2023-03-20", end_date="2023-03-26", aggregated_cases=99999),
    )
    mock_cache.get = AsyncMock(return_value=sentinel.model_dump_json())
    mock_cache.setex.reset_mock()

    response = client_with_cache.get("/cases/national", params={"now": "2023-03-26", "interval": 7})
    assert response.status_code == 200
    assert response.json()["data"]["aggregated_cases"] == 99999  # sentinel: came from cache
    assert response.headers.get("X-Cache-Status") == "HIT"
    mock_cache.setex.assert_not_called()


def test_trace_id_auto_generated(client):
    """No X-Trace-ID in request → middleware generates a UUID and echoes it in response."""
    import re
    UUID_RE = re.compile(r"^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$")
    response = client.get("/cases/national", params={"now": "2023-03-26", "interval": 7})
    trace_id = response.headers.get("X-Trace-ID")
    assert trace_id is not None
    assert UUID_RE.match(trace_id), f"Expected UUID, got: {trace_id}"


def test_trace_id_propagated(client):
    """X-Trace-ID provided in request → same value echoed in response."""
    custom_trace_id = "test-trace-abc-123"
    response = client.get(
        "/cases/national",
        params={"now": "2023-03-26", "interval": 7},
        headers={"X-Trace-ID": custom_trace_id},
    )
    assert response.headers.get("X-Trace-ID") == custom_trace_id


def test_unexpected_error_returns_500(client_broken):
    """Verifies global_exception_handler catches unexpected errors and returns 500."""
    response = client_broken.get("/cases/national", params={"now": "2023-03-26", "interval": 7})
    assert response.status_code == 500
    body = response.json()
    assert body["success"] is False
    assert body["message"] == "Internal server error."
