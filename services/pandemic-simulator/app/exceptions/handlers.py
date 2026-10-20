import logging

from fastapi import FastAPI, Request  # type: ignore
from fastapi.responses import JSONResponse  # type: ignore
from pydantic import ValidationError  # type: ignore

from exceptions.custom import EmptyDataError, ServiceValidationError
from utils.pydantic_model.response import APIResponse, ErrorModel

logger = logging.getLogger(__name__)


async def empty_data_error_handler(request: Request, exc: EmptyDataError):
    logger.error(f"Empty data error: {exc}")

    response = APIResponse(
        success=False,
        message="Empty data.",
        errors=ErrorModel(
            summary="No data available for the given date(s).",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=404,
    )


async def service_validation_error_handler(
    request: Request, exc: ServiceValidationError
):
    logger.error(f"Output data validation error in service: {exc}")

    if exc.errors:
        error_models = [
            ErrorModel(
                field=str(err["loc"][-1]) if err.get("loc") else None,
                summary=err.get("msg", "Output data validation failed."),
                detail=str(err),
            )
            for err in exc.errors
        ]
        errors = error_models if len(error_models) > 1 else error_models[0]
    else:
        errors = ErrorModel(
            summary="Output data validation failed.",
            detail=str(exc),
        )

    response = APIResponse(
        success=False,
        message="Validation error in the service.",
        errors=errors,
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=500,
    )


async def validation_error_handler(request: Request, exc: ValidationError):
    logger.error(f"Validation error: {exc}")

    if exc.errors():
        error_models = [
            ErrorModel(
                field=str(err["loc"][-1]) if err.get("loc") and len(err["loc"]) > 0 else None,
                summary=err.get("msg", "Input data validation failed."),
                detail=str(err),
            )
            for err in exc.errors()
        ]
        errors = error_models if len(error_models) > 1 else error_models[0]
    else:
        errors = ErrorModel(
            summary="Input data validation failed.",
            detail=str(exc),
        )

    response = APIResponse(
        success=False,
        message="Validation error.",
        errors=errors,
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
            summary="Unexpected error occurred during processing.",
            detail=str(exc),
        ),
    )

    return JSONResponse(
        content=response.model_dump(exclude_none=True),
        status_code=500,
    )


def register_exception_handlers(app: FastAPI):
    app.add_exception_handler(EmptyDataError, empty_data_error_handler)
    app.add_exception_handler(ServiceValidationError, service_validation_error_handler)
    app.add_exception_handler(ValidationError, validation_error_handler)
    app.add_exception_handler(Exception, global_exception_handler)
