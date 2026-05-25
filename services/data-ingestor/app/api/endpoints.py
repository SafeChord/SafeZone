# app/api/endpoints.py
import logging

from fastapi import APIRouter, Depends  # type: ignore

from utils.logging.baselogger import trace_id_var
from utils.pydantic_model.request import CovidDataModel
from utils.pydantic_model.response import APIResponse, HealthResponse

from core.settings import KAFKA_TOPIC
from api.dependencies import get_kafka_producer
from services.ingest_service import publish_event

router = APIRouter()
logger = logging.getLogger(__name__)


@router.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(
        success=True, message="Service is healthy.", status={"ingestor": "healthy"}
    )


@router.post("/covid_event", response_model=APIResponse)
async def covid_event_handler(
    payload: CovidDataModel,
    producer=Depends(get_kafka_producer),
):
    logger.info(
        "Received request to endpoint /covid_event with payload",
        extra={"event": "covid_event_request"},
    )

    await publish_event(
        producer=producer,
        topic=KAFKA_TOPIC,
        payload=payload,
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
