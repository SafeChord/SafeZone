# app/core/settings.py
import os
from dotenv import load_dotenv # type: ignore

# load environment variables
load_dotenv()

# Database settings
DB_URL = os.getenv("DB_URL", "sqlite:////db/test.db")

# Redis settings
REDIS_HOST = os.getenv("REDIS_HOST", "localhost")
REDIS_PORT = int(os.getenv("REDIS_PORT", "6379"))
REDIS_DB = int(os.getenv("REDIS_DB", "1"))
REDIS_PASSWORD = os.getenv("REDIS_PASSWORD", None)  # Optional, can be None
POLL_CACHE_VERSION_INTERVAL = int(os.getenv("POLL_CACHE_VERSION_INTERVAL", "60"))  # seconds

# Cache settings
CACHE_HOST = os.getenv("CACHE_HOST", "localhost")
CACHE_PORT = int(os.getenv("CACHE_PORT", "6479"))
CACHE_DB = int(os.getenv("CACHE_DB", "0"))
CACHE_PASSWORD = os.getenv("CACHE_PASSWORD", None)  # Optional, can be None

# Service settings
SERVICE_NAME = os.getenv("SERVICE_NAME", "AnalyticsAPI")
SERVICE_VERSION = os.getenv("SERVICE_VERSION", "0.0.0")
# Server settings
SERVER_IP = os.getenv("Server_IP", "0.0.0.0")
SERVER_PORT = int(os.getenv("Server_PORT", "8000"))
# Logging settings
LOG_LEVEL = os.getenv("LOG_LEVEL", "DEBUG")
