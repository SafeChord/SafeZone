# app/main.py
import uuid
import logging

import uvicorn  # type: ignore
from fastapi import FastAPI  # type: ignore

from utils.logging.baselogger import setup_logger
from utils.context import trace_id_var

from api.endpoints import router
from core.settings import SERVER_IP, SERVER_PORT, SERVICE_NAME, SERVICE_VERSION
from core.settings import LOG_LEVEL
from exceptions.handlers import register_exception_handlers


def create_app() -> FastAPI:
    """App factory: setup logging, routers, exception handlers, preload caches."""
    setup_logger(
        log_level=LOG_LEVEL, service_name=SERVICE_NAME, service_version=SERVICE_VERSION
    )
    app = FastAPI(
        title=SERVICE_NAME,
        version=SERVICE_VERSION,
    )
    app.include_router(router)
    register_exception_handlers(app)
    return app


app = create_app()
logger = logging.getLogger(__name__)


# ---- Middleware ----
@app.middleware("http")
async def add_trace_id(request, call_next):
    """
    Middleware to add trace_id to the request context.
    This is useful for logging and tracing requests.
    """
    trace_id = request.headers.get("X-Trace-ID", str(uuid.uuid4()))
    logger.debug(f"Received request with trace_id: {trace_id}")

    # Set the trace_id in the context variable
    token = trace_id_var.set(trace_id)

    try:
        response = await call_next(request)
    finally:
        trace_id_var.reset(token)
    response.headers["X-Trace-ID"] = trace_id
    return response


if __name__ == "__main__":
    logger.info(
        f"Starting {SERVICE_NAME} version {SERVICE_VERSION}",
        extra={"event": "service_startup"}
    )
    uvicorn.run("main:app", host=SERVER_IP, port=SERVER_PORT)
