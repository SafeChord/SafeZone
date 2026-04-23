# app/api/endpoints.py
import logging
from datetime import timedelta

from fastapi import APIRouter, Depends  # type: ignore

from utils.pydantic_model.request import (
    RegionParameters,
    CityParameters,
    NationalParameters,
)
from utils.pydantic_model.response import (
    AnalyticsAPIData,
    AnalyticsAPIResponse,
    HealthResponse,
)
from api.dependencies import (
    get_db_session,
    get_geo_cache,
    get_populations_cache,
    get_cache_client,
    get_cache_version,
)
from services.query_service import query_cases
from core.lifecycle import redis_cache


router = APIRouter()
logger = logging.getLogger(__name__)


@router.get("/health", response_model=HealthResponse)
async def health_check():
    return HealthResponse(success=True, message="Service is healthy.", status={"analytics-api": "healthy"})


@router.get("/cases/region", response_model=AnalyticsAPIResponse)
@redis_cache(endpoint="cases_region", ttl=86400)
async def process_region(
    params: RegionParameters = Depends(),
    db_session=Depends(get_db_session),
    geo_cache=Depends(get_geo_cache),
    populations_cache=Depends(get_populations_cache),
    cache_client=Depends(get_cache_client),
    cache_version=Depends(get_cache_version),
):
    end_date = params.now
    start_date = end_date - timedelta(days=int(params.interval) - 1)

    query_params = {
        "start_date": start_date,
        "end_date": end_date,
        "city": params.city,
        "region": params.region,
        "ratio": False if not params.ratio else True,
    }
    logger.info(
        f"Received region-level request to query data with params {query_params}.",
        extra={"event": "query_region_cases"},
    )

    query_result = query_cases(db_session, geo_cache, populations_cache, query_params)

    logger.info(
        f"Query region-level result: {query_result}",
        extra={"event": "query_region_cases"},
    )

    return AnalyticsAPIResponse(
        success=True,
        message="Data returned successfully",
        detail=f"Data returned successfully for dates {query_result['start_date']} ~ {query_result['end_date']}.",
        data=AnalyticsAPIData(**query_result),
    )


@router.get("/cases/city", response_model=AnalyticsAPIResponse)
@redis_cache(endpoint="cases_city", ttl=86400)
async def process_city(
    params: CityParameters = Depends(),
    db_session=Depends(get_db_session),
    geo_cache=Depends(get_geo_cache),
    populations_cache=Depends(get_populations_cache),
    cache_client=Depends(get_cache_client),
    cache_version=Depends(get_cache_version),
):
    end_date = params.now
    start_date = end_date - timedelta(days=int(params.interval) - 1)

    query_params = {
        "start_date": start_date,
        "end_date": end_date,
        "city": params.city,
        "ratio": False if not params.ratio else True,
    }

    logger.info(
        f"Received city-level request to query data with params {query_params}.",
        extra={"event": "query_city_cases"},
    )

    query_result = query_cases(db_session, geo_cache, populations_cache, query_params)

    logger.info(
        f"Query city-level result: {query_result}", extra={"event": "query_city_cases"}
    )

    return AnalyticsAPIResponse(
        success=True,
        message="Data returned successfully",
        detail=f"Data returned successfully for dates {query_result['start_date']} ~ {query_result['end_date']}.",
        data=AnalyticsAPIData(**query_result),
    )


@router.get("/cases/national", response_model=AnalyticsAPIResponse)
@redis_cache(endpoint="cases_national", ttl=86400)
async def process_national(
    params: NationalParameters = Depends(),
    db_session=Depends(get_db_session),
    geo_cache=Depends(get_geo_cache),
    populations_cache=Depends(get_populations_cache),
    cache_client=Depends(get_cache_client),
    cache_version=Depends(get_cache_version),
):
    end_date = params.now
    start_date = end_date - timedelta(days=int(params.interval) - 1)

    query_params = {
        "start_date": start_date,
        "end_date": end_date,
    }

    logger.info(
        f"Received national-level request to query data with params {query_params}.",
        extra={"event": "query_national_cases"},
    )

    query_result = query_cases(db_session, geo_cache, populations_cache, query_params)

    logger.info(
        f"Query national-level result: {query_result}",
        extra={"event": "query_national_cases"},
    )

    return AnalyticsAPIResponse(
        success=True,
        message="Data returned successfully",
        detail=f"Data returned successfully for dates {query_result['start_date']} ~ {query_result['end_date']}.",
        data=AnalyticsAPIData(**query_result),
    )
