# toolkit/time-server/api/endpoints.py
import json
import logging
from datetime import datetime, date, timedelta

import redis  # type: ignore
from fastapi import APIRouter  # type: ignore

from config.settings import REDIS_HOST, REDIS_PORT, REDIS_DB, REDIS_PASSWORD
from utils.pydantic_model.request import SetTimeModel
from utils.pydantic_model.response import APIResponse, SystemDateResponse
from utils.pydantic_model.response import MocktimeStatusResponse, MocktimeStatusData

router = APIRouter()
logger = logging.getLogger(__name__)

REDIS_PATH = "safezone:mock_date:config"

def get_redis_connection():
    """
    Create a Redis connection.
    """
    return redis.Redis(
        host=REDIS_HOST,
        port=REDIS_PORT,
        db=REDIS_DB,
        password=REDIS_PASSWORD,
        decode_responses=True,
    )

def get_mock_config():
    r = get_redis_connection()
    config = r.hgetall(REDIS_PATH)
    if not config:
        # If no config is found, set default values
        config = {
            "mock": "False",
            "mock_date": date.today().strftime("%Y-%m-%d"),
            "mock_update_time": datetime.now().replace(hour=0, minute=0, second=0, microsecond=0).strftime("%Y-%m-%d %H:%M:%S"),
            "acceleration": "1",
            "launch_time": datetime.now().strftime("%Y-%m-%d %H:%M:%S"),
        }
        r.hset(REDIS_PATH, mapping=config)
    return config


def get_system_date(
    mock_update_time: str, mock_date: str, acceleration: int = 1
) -> str:
    """
    Get the current date, adjusted by the number of bias days.
    """
    shift = datetime.now() - datetime.strptime(mock_update_time, "%Y-%m-%d %H:%M:%S")
    system_date = datetime.strptime(mock_date, "%Y-%m-%d") + shift * acceleration
    return system_date.strftime("%Y-%m-%d")


@router.get("/now", response_model=SystemDateResponse)
async def get_now():
    try:
        logger.debug("Received request to get_now model.")

        mock_config = get_mock_config()

        mock = bool(mock_config["mock"])

        if not mock:
            return SystemDateResponse(
                success=True,
                message="Current date retrieved successfully.",
                system_date=datetime.now().strftime("%Y-%m-%d"),
            )
        else:
            return SystemDateResponse(
                success=True,
                message="Mock date retrieved successfully.",
                system_date=get_system_date(
                    mock_update_time=mock_config["mock_update_time"],
                    mock_date=mock_config["mock_date"],
                    acceleration=int(mock_config["acceleration"]),
                ),
            )
    except Exception as e:
        logger.error(f"Error while get_now model: {str(e)}")
        return APIResponse(
            success=False,
            message="Error while get_now model.",
            detail=str(e),
        )


@router.post("/set", response_model=APIResponse)
async def set_mock_config(
    payload: SetTimeModel,
):
    try:
        logger.debug("Received request to set_time model.")

        data = payload.model_dump()

        logger.debug(f"Received data: {data}")
        # Ensure the Redis path exists and has default values if not set
        _ = get_mock_config()

        mock = data["mock"]
        # optional fields
        mock_date = data["mock_date"] if "mock_date" in data else None
        acceleration = data["acceleration"] if "acceleration" in data else None

        logger.debug(f"Setting mock: {mock}, mock_date: {mock_date}, acceleration: {acceleration}")
        r = get_redis_connection()
        if not mock:
            r.hset(REDIS_PATH, "mock", "False")
            r.hset(REDIS_PATH, "mock_date", date.today().strftime("%Y-%m-%d"))
            r.hset(
                REDIS_PATH,
                "mock_update_time",
                datetime.now().replace(hour=0, minute=0, second=0, microsecond=0).strftime("%Y-%m-%d %H:%M:%S"),
            )
            r.hset(REDIS_PATH, "acceleration", "1")
        else:
            if mock_date:
                r.hset(REDIS_PATH, "mock", "True")
                r.hset(REDIS_PATH, "mock_date", mock_date)
                r.hset(
                    REDIS_PATH,
                    "mock_update_time",
                    datetime.now().replace(hour=0, minute=0, second=0, microsecond=0).strftime("%Y-%m-%d %H:%M:%S"),
                )
            if acceleration:
                r.hset(REDIS_PATH, "acceleration", acceleration)
        
        logger.debug(f"Mock configuration updated successfully. {get_mock_config()}")

        return APIResponse(success=True, message="Time configuration set successfully.")
    except Exception as e:
        logger.error(f"Error while set_time model: {str(e)}")
        return APIResponse(
            success=False,
            message="Error while set_time model.",
            detail=str(e),
        )


@router.get("/status", response_model=MocktimeStatusResponse)
async def get_status():
    try:
        logger.debug("Received request to get_mock_config model.")

        mock_config = get_mock_config()
        mock = mock_config.get("mock", "False") == "True"
        # If mock is not enabled, return the current date and system date
        if not mock:
            return MocktimeStatusResponse(
                success=True,
                message="Time server is not in mock mode.",
                data=MocktimeStatusData(
                    mock=mock,
                    mock_date=datetime.now().strftime("%Y-%m-%d"),
                    mock_update_time=mock_config.get(
                        "mock_update_time", "1970-01-01 00:00:00"
                    ),
                    launch_time=mock_config.get("launch_time", "1970-01-01 00:00:00"),
                    acceleration=1,
                    system_date=datetime.now().strftime("%Y-%m-%d"),
                ),
            )
        return MocktimeStatusResponse(
            success=True,
            message="Time configuration retrieved successfully.",
            data=MocktimeStatusData(
                mock=mock,
                mock_date=mock_config.get("mock_date", "1970-01-01"),
                mock_update_time=mock_config.get(
                    "mock_update_time", "1970-01-01 00:00:00"
                ),
                launch_time=mock_config.get("launch_time", "1970-01-01 00:00:00"),
                acceleration=mock_config.get("acceleration", "1"),
                system_date=get_system_date(
                    mock_update_time=mock_config.get("mock_update_time"),
                    mock_date=mock_config.get("mock_date"),
                    acceleration=int(mock_config.get("acceleration", 1)),
                ),
            ),
        )
    except Exception as e:
        logger.error(f"Error while get_mock_config model: {str(e)}")
        return APIResponse(
            success=False,
            message="Error while get_mock_config model.",
            detail=str(e),
        )
