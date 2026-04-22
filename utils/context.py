import contextvars
from typing import Optional

trace_id_var: contextvars.ContextVar[str] = contextvars.ContextVar("trace_id", default="-")
cache_status_var: contextvars.ContextVar[Optional[str]] = contextvars.ContextVar("cache_status", default=None)