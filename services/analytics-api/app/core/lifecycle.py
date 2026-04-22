# app/core/lifecycle.py
import json
import logging
import hashlib
import asyncio
import functools

import redis.asyncio as aioredis  # type: ignore

from utils.pydantic_model.response import AnalyticsAPIResponse
from core.context import cache_status_var
from core.settings import REDIS_HOST, REDIS_PORT, REDIS_DB, REDIS_PASSWORD
from core.settings import CACHE_HOST, CACHE_PORT, CACHE_DB, CACHE_PASSWORD
from core.settings import POLL_CACHE_VERSION_INTERVAL


logger = logging.getLogger(__name__)


async def get_redis_client(redis_name):
    try:
        if redis_name == "redis-state":
            redis_client = aioredis.Redis(
                host=REDIS_HOST,
                port=REDIS_PORT,
                db=REDIS_DB,
                password=REDIS_PASSWORD,
                decode_responses=True,
            )
            logger.debug("Connected to redis-state successfully.")
        if redis_name == "redis-cache":
            redis_client = aioredis.Redis(
                host=CACHE_HOST,
                port=CACHE_PORT,
                db=CACHE_DB,
                password=CACHE_PASSWORD,
                decode_responses=True,
            )
            logger.debug("Connected to redis-cache successfully.")
        await redis_client.ping()  # Test the connection
        return redis_client
    except Exception as e:
        logger.error(f"Failed to connect to Redis: {e}")
        return None


async def poll_cache_version(app_state):
    """
    A background task that periodically polls redis-state for the cache version.
    """
    while True:
        await asyncio.sleep(POLL_CACHE_VERSION_INTERVAL)
        try:
            version = await app_state.redis_client.get(
                "current_cache_version"
            )
            if version and version != app_state.cache_version:
                app_state.cache_version = version
        except Exception as e:
            logger.error(f"Error polling cache version: {e}")


def generate_cache_key(cache_version: str, endpoint: str, params_model):
    # sort by keys to ensure consistent hashing
    raw = json.dumps(params_model.model_dump(), sort_keys=True, default=str)
    hashed = hashlib.sha256(raw.encode()).hexdigest()
    return f"{cache_version}:{endpoint}:{hashed}"


def redis_cache(endpoint, ttl=86400):
    # Per-endpoint lock dict — prevents cache stampede within a single process.
    # Keys are cache_key strings; unbounded growth is acceptable given analytics
    # key space is bounded (date × interval × region).
    # TODO: replace with Redis NX distributed lock when scaling to multi-replica.
    _locks: dict[str, asyncio.Lock] = {}

    def decorator(func):
        @functools.wraps(func)
        async def wrapper(*args, **kwargs):
            # dependencies injected via Depends() — no Request access needed
            cache_client = kwargs.get("cache_client", None)
            cache_version = kwargs.get("cache_version", "")
            params = kwargs.get("params", None)

            if not cache_client or not params:
                logger.warning("Cache client is not available, skipping cache.")
                return await func(*args, **kwargs)

            cache_key = generate_cache_key(cache_version, endpoint, params)
            logger.debug(f"Generated cache key: {cache_key}")

            # Fast path: check cache without lock (HIT avoids serialization)
            cached = await cache_client.get(cache_key)
            if cached:
                logger.debug(f"Cache hit: {cache_key}")
                cache_status_var.set("HIT")
                return AnalyticsAPIResponse.model_validate_json(cached)

            # Slow path: acquire per-key lock to prevent stampede on MISS
            if cache_key not in _locks:
                _locks[cache_key] = asyncio.Lock()
            async with _locks[cache_key]:
                # Double-check: another coroutine may have written cache while we waited
                cached = await cache_client.get(cache_key)
                if cached:
                    logger.debug(f"Cache hit (post-lock): {cache_key}")
                    cache_status_var.set("HIT")
                    return AnalyticsAPIResponse.model_validate_json(cached)

                logger.debug(f"Cache miss: {cache_key}")
                cache_status_var.set("MISS")
                resp = await func(*args, **kwargs)
                if getattr(resp, "success", True) and isinstance(resp, AnalyticsAPIResponse):
                    await cache_client.setex(
                        cache_key, ttl, resp.model_dump_json(exclude="timestamp")
                    )
                return resp

        return wrapper

    return decorator
