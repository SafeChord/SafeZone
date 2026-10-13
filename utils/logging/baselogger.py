import sys
import time
import logging
from logging import StreamHandler
from datetime import datetime, timezone

from pythonjsonlogger.json import JsonFormatter  # type: ignore

from utils.context import trace_id_var

log_fields = [
    "timestamp",
    "ts",
    "levelname",
    "service",
    "service_version",
    "module",
    "event",
    "trace_id",
    "message",
    "extra",
]


class ServiceInfoFilter(logging.Filter):
    def __init__(self, service_name, service_version):
        super().__init__()
        self.service_name = service_name
        self.service_version = service_version

    def filter(self, record):
        record.ts = time.time()
        record.timestamp = datetime.now(timezone.utc).isoformat().replace("+00:00", "Z")
        record.trace_id = trace_id_var.get()
        record.service = self.service_name
        record.service_version = self.service_version
        return True


def setup_logger(log_level: str, service_name: str, service_version: str, log_output: str = "stdout"):
    logger = logging.getLogger()

    if logger.hasHandlers():
        logger.debug("Logger already configured, skipping setup.")
        return
    
    logger.setLevel(log_level)

    if log_output == "stdout":
        json_handler = StreamHandler(sys.stdout)
    elif log_output == "stderr":
        json_handler = StreamHandler(sys.stderr)

    json_formatter = JsonFormatter(
        fmt=" ".join([f"%({field})s" for field in log_fields]),
        # To ensure cross-language log compatibility with tools like Loki,
        # this logger is configured to replicate the field structure of the
        # popular Golang Zap logger (e.g., 'level', 'msg').
        rename_fields={"levelname": "level", "message": "msg"},
    )
    json_handler.setFormatter(json_formatter)
    json_handler.addFilter(ServiceInfoFilter(service_name, service_version))
    logger.addHandler(json_handler)

    logger.addFilter(ServiceInfoFilter(service_name, service_version))
