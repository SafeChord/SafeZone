"""Requirement tests for the data-ingestor blueprint (ING-R4, ING-R5).

Each test states a promise the rest of the system relies on. Do not edit, loosen
or delete them to make an implementation pass; raise an objection instead.
"""
import asyncio
import time
from unittest.mock import AsyncMock, patch

import httpx  # type: ignore
import pytest  # type: ignore
from aiokafka.errors import KafkaConnectionError, KafkaTimeoutError  # type: ignore
from fastapi.testclient import TestClient  # type: ignore

from api.dependencies import get_kafka_producer
from main import create_app

SUCCESS_MESSAGE = "Data produced to Kafka successfully."
VALID_RECORD = {"date": "2023-03-20", "city": "台北市", "region": "信義區", "cases": 50}


def post_record(producer, record):
    with patch("main.startup_kafka", new_callable=AsyncMock, return_value=producer):
        with patch("main.shutdown_kafka", new_callable=AsyncMock):
            app = create_app()
            app.dependency_overrides[get_kafka_producer] = lambda: producer
            # The caller sees the HTTP response, whichever layer builds it.
            with TestClient(app, raise_server_exceptions=False) as client:
                return client.post("/covid_event", json=record)


def failing_producer(error):
    producer = AsyncMock()
    producer.send_and_wait.side_effect = error
    producer.send.side_effect = error
    return producer


# ING-R4: success means the topic has the event; a failed publish is reported as a failure.
@pytest.mark.parametrize(
    "producer",
    [
        None,
        failing_producer(KafkaConnectionError("broker unreachable")),
        failing_producer(KafkaTimeoutError("no ack from broker")),
    ],
    ids=["producer unavailable", "publish raises connection error", "publish times out"],
)
def test_ing_r4_failed_publish_is_reported_as_failure(producer):
    resp = post_record(producer, VALID_RECORD)

    assert 500 <= resp.status_code < 600
    body = resp.json()
    assert body["success"] is False
    assert body.get("message") != SUCCESS_MESSAGE


def unacknowledging_producer():
    async def never_completes(*args, **kwargs):
        await asyncio.Event().wait()

    producer = AsyncMock()
    producer.send_and_wait.side_effect = never_completes
    producer.send.side_effect = never_completes
    return producer


# ING-R4: a publish the topic never acknowledges fails within 4 seconds (decided on #82),
# below the simulator's 5-second client timeout, so the caller is told it failed.
# A broker lost after startup looks like this: the producer retries and never raises.
@pytest.mark.asyncio
async def test_ing_r4_unacknowledged_publish_fails_within_bound():
    producer = unacknowledging_producer()
    app = create_app()
    app.dependency_overrides[get_kafka_producer] = lambda: producer
    transport = httpx.ASGITransport(app=app, raise_app_exceptions=False)

    async with httpx.AsyncClient(transport=transport, base_url="http://ingestor") as client:
        start = time.monotonic()
        # The outer limit only keeps a hanging implementation from hanging the suite.
        resp = await asyncio.wait_for(client.post("/covid_event", json=VALID_RECORD), timeout=10)
        elapsed = time.monotonic() - start

    assert 500 <= resp.status_code < 600
    assert resp.json()["success"] is False
    assert elapsed < 4.5


# ING-R5: an invalid record is rejected and nothing is published.
@pytest.mark.parametrize(
    "field, value",
    [("date", "2026-13-45"), ("date", "2026-02-30"), ("cases", 2147483648)],
    ids=["month out of range", "day not in month", "cases above int32"],
)
def test_ing_r5_invalid_record_is_rejected_and_not_published(field, value):
    producer = AsyncMock()

    resp = post_record(producer, {**VALID_RECORD, field: value})

    assert resp.status_code == 422
    assert not producer.mock_calls
