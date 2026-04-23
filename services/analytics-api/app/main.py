# app/main.py
import asyncio
import uuid
import logging
from contextlib import asynccontextmanager

import uvicorn  # type: ignore
from fastapi import FastAPI  # type: ignore
from sqlalchemy import create_engine  # type: ignore
from sqlalchemy.orm import sessionmaker  # type: ignore
from starlette.datastructures import MutableHeaders  # type: ignore
from starlette.types import ASGIApp, Receive, Scope, Send  # type: ignore

from utils.logging.baselogger import setup_logger
from utils.context import trace_id_var
from core.context import cache_status_var

from api.endpoints import router
from core.settings import SERVER_IP, SERVER_PORT, SERVICE_NAME, SERVICE_VERSION
from core.settings import LOG_LEVEL, DB_URL
from exceptions.handlers import register_exception_handlers
from core.lifecycle import get_redis_client, poll_cache_version
from services.cache_service import get_city_region_cache, get_populations_cache


logger = logging.getLogger(__name__)


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


class TraceAndCacheMiddleware:
    """Pure ASGI middleware for trace ID propagation and cache status header injection.

    Uses pure ASGI (not BaseHTTPMiddleware) so the handler runs in the same
    coroutine context — ContextVar changes made by the cache decorator are
    visible here without child-task isolation issues.
    """

    def __init__(self, app: ASGIApp) -> None:
        self.app = app

    async def __call__(self, scope: Scope, receive: Receive, send: Send) -> None:
        if scope["type"] != "http":
            await self.app(scope, receive, send)
            return

        raw_headers = dict(scope.get("headers", []))
        trace_id = raw_headers.get(b"x-trace-id", b"").decode() or str(uuid.uuid4())
        logger.debug(f"Received request with trace_id: {trace_id}")

        trace_token = trace_id_var.set(trace_id)
        cache_token = cache_status_var.set(None)

        async def send_with_extra_headers(message: dict) -> None:
            if message["type"] == "http.response.start":
                headers = MutableHeaders(scope=message)
                headers["x-trace-id"] = trace_id
                cache_status = cache_status_var.get()
                if cache_status:
                    headers["x-cache-status"] = cache_status
            await send(message)

        try:
            await self.app(scope, receive, send_with_extra_headers)
        finally:
            trace_id_var.reset(trace_token)
            cache_status_var.reset(cache_token)


def create_app() -> FastAPI:
    """App factory: setup logging, routers, exception handlers, middleware."""
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
    app.add_middleware(TraceAndCacheMiddleware)
    return app


app = create_app()


if __name__ == "__main__":
    logger.info(
        f"Starting {SERVICE_NAME} version {SERVICE_VERSION}",
        extra={"event": "service_startup"},
    )
    uvicorn.run("main:app", host=SERVER_IP, port=SERVER_PORT)
