import json
import time
import logging

from pydantic import BaseModel  # type: ignore

from utils.pydantic_model.request import CovidDataModel


logger = logging.getLogger(__name__)


# Kafka event contract model
# Contract file: utils.contracts.covid_event.json
class CovidContract(BaseModel):
    event_type: str
    event_time: int
    trace_id: str
    payload: CovidDataModel
    version: str


def generate_partition_key(city: str, region: str) -> str:
    return f"{city}-{region}"


async def publish_event(producer, topic: str, payload: CovidDataModel, trace_id: str = "-"):
    if not producer:
        logger.error(
            "Kafka producer is not available, cannot send data to Kafka.",
            extra={"event": "kafka_producer_not_available"},
        )
        return

    partition_key = generate_partition_key(city=payload.city, region=payload.region)

    event = CovidContract(
        event_type="covid_event",
        event_time=int(time.time() * 1000),
        trace_id=trace_id,
        payload=payload,
        version="0.1.0",
    )
    logger.debug(
        f"Sending event to Kafka: with payload {json.dumps(event.model_dump())} and partition key {partition_key}",
        extra={"event": "send_event_to_kafka"},
    )

    await producer.send_and_wait(
        topic=topic,
        value=json.dumps(event.model_dump()).encode("utf-8"),
        key=partition_key.encode("utf-8"),
    )
