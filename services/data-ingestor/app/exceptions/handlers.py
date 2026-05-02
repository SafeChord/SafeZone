import logging

from fastapi import FastAPI, Request  # type: ignore
from fastapi.responses import JSONResponse  # type: ignore
from pydantic import ValidationError  # type: ignore

from utils.pydantic_model.response import ErrorModel, APIResponse

logger = logging.getLogger(__name__)


async def validation_error_handler(request: Request, exc: ValidationError):
    logger.error(f"Validation error: {exc}")

    resp = APIResponse(
        success=False,
        message="Validation error occurred",
        errors=ErrorModel(
            field=exc.errors()[0].get("loc", ["unknown"])[0],
            summary=exc.errors()[0].get("msg", "Invalid input"),
            detail=str(exc),
        ),
    )
    return JSONResponse(
        status_code=422,
        content=resp.model_dump(),
    )


async def global_exception_handler(request: Request, exc: Exception):
    logger.error(f"Error processing request: {exc}")

    resp = APIResponse(
        success=False,
        message="Failed to process the request",
        errors=ErrorModel(
            field="unknown",
            summary="Error processing the request, please try again later.",
            detail=str(exc),
        ),
    )
    return JSONResponse(
        status_code=500,
        content=resp.model_dump(),
    )


def register_exception_handlers(app: FastAPI):
    app.add_exception_handler(ValidationError, validation_error_handler)
    app.add_exception_handler(Exception, global_exception_handler)
