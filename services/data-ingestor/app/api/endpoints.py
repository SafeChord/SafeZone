# app/api/endpoints.py
import json
import time
import logging

from fastapi import APIRouter, Request  # type: ignore
from fastapi.responses import JSONResponse  # type: ignore
from pydantic import BaseModel, ValidationError  # type: ignore

from utils.logging.baselogger import trace_id_var
from utils.pydantic_model.request import CovidDataModel
from utils.pydantic_model.response import ErrorModel, APIResponse, HealthResponse

from config.settings import KAFKA_TOPIC

router = APIRouter()
logger = logging.getLogger(__name__)


# Kafka event contract model
# This model defines the structure of the event that will be sent to Kafka.
# Contract file: utils.contracts.covid_event.json
class CovidContract(BaseModel):
    event_type: str
    event_time: int
    trace_id: str
    payload: CovidDataModel
    version: str


def generate_partition_key(city: str, region: str) -> str:
    return f"{city}-{region}"


async def sent_to_kafka(
    producer, payload: CovidDataModel, partition_key: str, trace_id: str = "-"
):
    if not producer:
        logger.error(
            "Kafka producer is not available, cannot send data to Kafka.",
            extra={"event": "kafka_producer_not_available"},
        )
        return

    event = CovidContract(
        event_type="covid_event",
        event_time=int(time.time() * 1000),  # current time in milliseconds
        trace_id=trace_id,
        payload=payload,  # Ensure payload matches the model
        version="0.1.0",
    )
    logger.debug(
        f"Sending event to Kafka: with payload {json.dumps(event.model_dump())} and partition key {partition_key}",
        extra={"event": "send_event_to_kafka"},
    )

    await producer.send_and_wait(
        topic=KAFKA_TOPIC,
        value=json.dumps(event.model_dump()).encode("utf-8"),
        key=partition_key.encode("utf-8"),
    )


@router.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(
        success=True, message="Service is healthy.", status={"ingestor": "healthy"}
    )


@router.post("/covid_event", response_model=APIResponse)
async def covid_event_handler(
    request: Request,
    payload: CovidDataModel,
):
    try:
        logger.info(
            "Received request to endpoint /covid_event with payload",
            extra={"event": "covid_event_request"},
        )

        data = payload.model_dump()  # Convert Pydantic model to dict
        await sent_to_kafka(
            producer=request.app.state.kafka_producer,
            payload=payload,
            partition_key=generate_partition_key(
                city=data["city"],
                region=data["region"],
            ),
            trace_id=trace_id_var.get(),
        )
        logger.info(
            "Data produced to Kafka successfully.",
            extra={"event": "data_produced_to_kafka"},
        )

        return APIResponse(
            success=True,
            message="Data produced to Kafka successfully.",
        )
    except ValidationError as ve:
        logger.error(f"Validation error: {ve}")
        resp = APIResponse(
            success=False,
            message="Validation error occurred",
            errors=ErrorModel(
                field=ve.errors()[0].get("loc", ["unknown"])[0],
                summary=ve.errors()[0].get("msg", "Invalid input"),
                detail=str(ve),
            ),
        )
        return JSONResponse(
            status_code=422,
            content=resp.model_dump(),
        )
    except Exception as e:
        logger.error(f"Error processing request: {e}")
        resp = APIResponse(
            success=False,
            message="Failed to process the request",
            errors=ErrorModel(
                field="unknown",
                summary="Error processing the request, please try again later.",
                detail=str(e),
            ),
        )
        return JSONResponse(
            status_code=500,
            content=resp.model_dump(),
        )
