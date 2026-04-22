import contextvars

trace_id_var: contextvars.ContextVar[str] = contextvars.ContextVar("trace_id", default="-")