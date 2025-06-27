import sys
import logging
import contextvars
from logging import StreamHandler

from pythonjsonlogger.json import JsonFormatter # type: ignore 

trace_id_var = contextvars.ContextVar("trace_id", default="-")
log_fields = [
    "timestamp", "levelname", "service", "service_version", "module", "event", "trace_id", "message", "extra"
]

class ServiceInfoFilter(logging.Filter):
    def __init__(self, service_name, service_version):
        super().__init__()
        self.service_name = service_name
        self.service_version = service_version

    def filter(self, record):
        record.trace_id = trace_id_var.get()
        record.service = self.service_name
        record.service_version = self.service_version
        return True

def setup_logger(log_level: str, service_name: str, service_version: str):
    logger = logging.getLogger()
    logger.setLevel(log_level)

    json_handler = StreamHandler(sys.stdout)
    json_formatter = JsonFormatter(
        fmt=' '.join([f'%({field})s' for field in log_fields]),
        rename_fields={"asctime": "timestamp"},
    )
    json_handler.setFormatter(json_formatter)
    json_handler.addFilter(ServiceInfoFilter(service_name, service_version))
    logger.addHandler(json_handler)
    
    logger.addFilter(ServiceInfoFilter(service_name, service_version))