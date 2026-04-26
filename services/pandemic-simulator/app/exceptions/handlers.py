import logging

from fastapi import FastAPI, Request  # type: ignore
from fastapi.responses import JSONResponse  # type: ignore
from pydantic import ValidationError  # type: ignore

from utils.pydantic_model.response import ErrorModel, APIResponse
from exceptions.custom import (
    InvalidDateRangeError,
    APIValidationError,
    ServiceValidationError,
    EmptyDataError,
)

logger = logging.getLogger(__name__)


async def invalid_date_range_error_handler(
    request: Request, exc: InvalidDateRangeError
):
    logger.error(f"Invalid date range error: {exc}")

    response = APIResponse(
        success=False,
        message="Invalid date range.",
        errors=ErrorModel(
            fields="date range",
            summary="the 'start_date' must be before the 'end_date'.",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=400,
    )


async def empty_data_error_handler(request: Request, exc: EmptyDataError):
    logger.error(f"Empty data error: {exc}")

    response = APIResponse(
        success=False,
        message="Empty data.",
        errors=ErrorModel(
            fields="data",
            summary="No data available for the given date(s).",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=500,
    )


async def service_validation_error_handler(
    request: Request, exc: ServiceValidationError
):
    logger.error(f"Output data validation error in service: {exc}")

    response = APIResponse(
        success=False,
        message="Validation error in the service.",
        errors=ErrorModel(
            fields="output data",
            summary="Output data validation failed.",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=500,
    )


async def api_validation_error_handler(request: Request, exc: APIValidationError):
    logger.error(f"API validation error: {exc}")

    response = APIResponse(
        success=False,
        message="Validation error in the API.",
        errors=ErrorModel(
            fields="input data",
            summary="Input data validation failed.",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=422,
    )


async def validation_error_handler(request: Request, exc: ValidationError):
    logger.error(f"Validation error: {exc}")

    response = APIResponse(
        success=False,
        message="Validation error.",
        errors=ErrorModel(
            fields="input data",
            summary="Input data validation failed.",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=422,
    )


async def global_exception_handler(request: Request, exc: Exception):
    logger.error(f"Internal server error: {exc}")

    response = APIResponse(
        success=False,
        message="Internal server error.",
        errors=ErrorModel(
            fields="unknown",
            summary="Unexpected error occurred during processing.",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=500,
    )


# Register the exception handlers
def register_exception_handlers(app: FastAPI):
    # in service exception handlers
    app.add_exception_handler(InvalidDateRangeError, invalid_date_range_error_handler)
    app.add_exception_handler(EmptyDataError, empty_data_error_handler)
    app.add_exception_handler(ServiceValidationError, service_validation_error_handler)
    app.add_exception_handler(APIValidationError, api_validation_error_handler)
    # global exception handler
    app.add_exception_handler(
        ValidationError, api_validation_error_handler
    )  # handle pydantic validation errors globally
    app.add_exception_handler(Exception, global_exception_handler)
