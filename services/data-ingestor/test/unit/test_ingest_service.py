# test/unit/test_ingest_service.py
import json
import logging

import pytest  # type: ignore
from jsonschema import Draft202012Validator  # type: ignore
from unittest.mock import AsyncMock

from exceptions.custom import KafkaUnavailableError
from utils.pydantic_model.request import CovidDataModel
from services.ingest_service import generate_partition_key, publish_event


@pytest.fixture(scope="module")
def logger():
    return logging.getLogger(__name__)


# Import test cases
with open("/test/cases/test_ingest_service.json", encoding="utf-8") as f:
    test_cases = json.load(f)


partition_key_cases = [c for c in test_cases if c["component_under_test"] == "generate_partition_key"]
publish_event_cases = [c for c in test_cases if c["component_under_test"] == "publish_event"]
no_producer_cases = [c for c in test_cases if c["component_under_test"] == "publish_event_no_producer"]


@pytest.mark.parametrize("case", partition_key_cases, ids=lambda c: c["test_describes"])
def test_generate_partition_key(case, logger):
    result = generate_partition_key(city=case["city"], region=case["region"])
    assert result == case["expected"]
    logger.debug(f"Partition key: {result}")


@pytest.mark.asyncio
@pytest.mark.parametrize("case", publish_event_cases, ids=lambda c: c["test_describes"])
async def test_publish_event(case, logger):
    mock_producer = AsyncMock()
    payload = CovidDataModel(**case["payload"])

    await publish_event(
        producer=mock_producer,
        topic=case["topic"],
        payload=payload,
        trace_id=case["trace_id"],
    )

    mock_producer.send_and_wait.assert_called_once()
    call_kwargs = mock_producer.send_and_wait.call_args[1]
    assert call_kwargs["topic"] == case["topic"]
    assert call_kwargs["key"] == case["expected_partition_key"].encode("utf-8")
    logger.debug("publish_event called send_and_wait correctly.")


@pytest.mark.asyncio
@pytest.mark.parametrize("case", no_producer_cases, ids=lambda c: c["test_describes"])
async def test_publish_event_no_producer(case, logger):
    payload = CovidDataModel(**case["payload"])

    # ING-R4: Should raise KafkaUnavailableError when producer is unavailable
    with pytest.raises(KafkaUnavailableError):
        await publish_event(
            producer=None,
            topic=case["topic"],
            payload=payload,
            trace_id=case["trace_id"],
        )
    logger.debug("publish_event raised KafkaUnavailableError when producer is None.")


# ING-R2: every event the ingestor publishes satisfies the event contract.
# Format checking is on, so `format: date` is enforced and not just an annotation.
@pytest.mark.asyncio
@pytest.mark.parametrize("case", publish_event_cases, ids=lambda c: c["test_describes"])
async def test_ing_r2_published_event_satisfies_contract(case):
    with open("/contract/covid_event.json", encoding="utf-8") as f:
        validator = Draft202012Validator(
            json.load(f), format_checker=Draft202012Validator.FORMAT_CHECKER
        )
    mock_producer = AsyncMock()

    await publish_event(
        producer=mock_producer,
        topic=case["topic"],
        payload=CovidDataModel(**case["payload"]),
        trace_id=case["trace_id"],
    )

    event = json.loads(mock_producer.send_and_wait.call_args[1]["value"])
    errors = [e.message for e in validator.iter_errors(event)]
    assert not errors, errors
