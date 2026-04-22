# app/core/context.py
import contextvars
from typing import Optional

# Request-scoped cache status for HTTP response header decoration (X-Cache-Status).
# This is analytics-api specific — not a shared cross-cutting concern.
# TODO: replace with Redis NX distributed lock when scaling to multi-replica.
cache_status_var: contextvars.ContextVar[Optional[str]] = contextvars.ContextVar(
    "cache_status", default=None
)
