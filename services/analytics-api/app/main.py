# app/main.py
import asyncio
import uuid
import logging
from contextlib import asynccontextmanager

import uvicorn  # type: ignore
from fastapi import FastAPI  # type: ignore
from sqlalchemy import create_engine  # type: ignore
from sqlalchemy.orm import sessionmaker  # type: ignore

from utils.logging.baselogger import setup_logger 
from utils.context import trace_id_var

from api.endpoints import router
from config.settings import SERVER_IP, SERVER_PORT, SERVICE_NAME, SERVICE_VERSION
from config.settings import LOG_LEVEL, DB_URL
from exceptions.handlers import register_exception_handlers
from config.cache import get_city_region_cache, get_populations_cache, get_redis_client, poll_cache_version


@asynccontextmanager
async def lifespan(app: FastAPI):
    engine = create_engine(DB_URL)
    app.state.engine = engine
    app.state.Session = sessionmaker(bind=engine)
    app.state.cache_client = await get_redis_client("redis-cache")
    app.state.redis_client = await get_redis_client("redis-state")
    app.state.city_region_cache = get_city_region_cache(app.state.Session)
    app.state.populations_cache = get_populations_cache(app.state.Session)
    
    # Start the cache version poller
    app.state.cache_version = str(uuid.uuid4())
    poller_task = asyncio.create_task(poll_cache_version(app.state))

    yield
    # Cleanup
    poller_task.cancel()
    engine.dispose()


def create_app() -> FastAPI:
    """App factory: setup logging, routers, exception handlers, preload caches."""
    setup_logger(
        log_level=LOG_LEVEL, service_name=SERVICE_NAME, service_version=SERVICE_VERSION
    )
    app = FastAPI(
        title=SERVICE_NAME,
        version=SERVICE_VERSION,
        lifespan=lifespan,
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

    # Forward cache status from redis_cache decorator
    cache_status = getattr(request.state, "cache_status", None)
    if cache_status:
        response.headers["X-Cache-Status"] = cache_status

    return response


if __name__ == "__main__":
    logger.info(
        f"Starting {SERVICE_NAME} version {SERVICE_VERSION}",
        extra={"event": "service_startup"},
    )
    uvicorn.run("main:app", host=SERVER_IP, port=SERVER_PORT)
