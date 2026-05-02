# app/config/settings.py
import os
from dotenv import load_dotenv # type: ignore

# load environment variables
load_dotenv()

# Service settings
SERVICE_NAME = os.getenv("SERVICE_NAME", "CovidDataSimulator")
SERVICE_VERSION = os.getenv("SERVICE_VERSION", "0.1.0")

# Server settings
SERVER_IP = os.getenv("SERVER_IP", "0.0.0.0")
SERVER_PORT = int(os.getenv("SERVER_PORT", "8000"))

# Downstream service settings
MAX_CONCURRENT_REQUESTS = int(os.getenv("MAX_CONCURRENT_REQUESTS", "10"))
INGESTOR_URL = os.getenv("INGESTOR_URL", "http://0.0.0.0:8001")

# Logging settings
LOG_LEVEL = os.getenv("LOG_LEVEL", "INFO")