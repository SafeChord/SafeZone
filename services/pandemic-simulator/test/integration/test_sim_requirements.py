"""Requirement tests for the pandemic-simulator blueprint (SIM-R4 to SIM-R7).

Each test states a promise the rest of the system relies on. Do not edit, loosen
or delete them to make an implementation pass; raise an objection instead.

The source file is the test fixture `data/test_data.csv`: 2023-03-20 holds two
valid records, 2023-07-01 holds one valid record and one whose summed cases are
0, which breaks the ingest request rules, and 2025 holds no rows.
"""
import httpx  # type: ignore
import pytest  # type: ignore
import respx  # type: ignore
from fastapi.testclient import TestClient  # type: ignore

from core.settings import INGESTOR_URL  # type: ignore
from main import create_app  # type: ignore

INGEST_URL = INGESTOR_URL + "/covid_event"
DATE_WITH_TWO_RECORDS = "2023-03-20"
DATE_WITH_INVALID_RECORD = "2023-07-01"


@pytest.fixture
def client():
    # The caller sees the HTTP response, whichever layer builds it.
    with TestClient(create_app(), raise_server_exceptions=False) as c:
        yield c


@pytest.fixture
def ingest():
    # Any request outside the mocked ingest route fails the test.
    with respx.mock(assert_all_called=False) as router:
        yield router


def error_entries(body):
    errors = body.get("errors")
    if errors is None:
        return []
    return errors if isinstance(errors, list) else [errors]


def assert_reports_failure(resp):
    assert resp.status_code >= 400
    assert resp.json()["success"] is False


# SIM-R5: success means every record was accepted.
@pytest.mark.parametrize("error_status", [422, 500, 503])
def test_sim_r5_ingest_error_on_one_record_fails_the_trigger(client, ingest, error_status):
    route = ingest.post(INGEST_URL).mock(
        side_effect=[httpx.Response(200, json={}), httpx.Response(error_status, json={})]
    )

    resp = client.get(f"/simulate/daily?date={DATE_WITH_TWO_RECORDS}")

    assert route.call_count == 2
    assert_reports_failure(resp)


# SIM-R5: an ingest request that never gets an answer is not an accepted record.
def test_sim_r5_unreachable_ingest_fails_the_trigger(client, ingest):
    ingest.post(INGEST_URL).mock(side_effect=httpx.ConnectError("ingest unreachable"))

    resp = client.get(f"/simulate/daily?date={DATE_WITH_TWO_RECORDS}")

    assert_reports_failure(resp)
    # An unexpected failure has no offending field.
    assert all("field" not in e for e in error_entries(resp.json()))


# SIM-R4: when one record breaks the ingest request rules, none is sent.
def test_sim_r4_invalid_record_sends_nothing_and_names_the_field(client, ingest):
    route = ingest.post(INGEST_URL).mock(return_value=httpx.Response(200, json={}))

    resp = client.get(f"/simulate/daily?date={DATE_WITH_INVALID_RECORD}")

    assert route.call_count == 0
    assert_reports_failure(resp)
    assert "cases" in [e.get("field") for e in error_entries(resp.json())]


# SIM-R6: a trigger with no data sends nothing and fails (404 per #84).
@pytest.mark.parametrize(
    "path",
    [
        "/simulate/daily?date=2025-03-20",
        "/simulate/interval?start_date=2025-03-20&end_date=2025-04-05",
    ],
    ids=["daily", "interval"],
)
def test_sim_r6_no_data_sends_nothing_and_answers_404(client, ingest, path):
    route = ingest.post(INGEST_URL).mock(return_value=httpx.Response(200, json={}))

    resp = client.get(path)

    assert route.call_count == 0
    assert resp.status_code == 404
    body = resp.json()
    assert body["success"] is False
    # No data is not the fault of any field.
    assert all("field" not in e for e in error_entries(body))


# SIM-R7: an interval whose end precedes its start is rejected as invalid.
def test_sim_r7_end_before_start_is_rejected_and_sends_nothing(client, ingest):
    route = ingest.post(INGEST_URL).mock(return_value=httpx.Response(200, json={}))

    resp = client.get("/simulate/interval?start_date=2023-03-25&end_date=2023-03-20")

    assert route.call_count == 0
    assert resp.status_code == 422
