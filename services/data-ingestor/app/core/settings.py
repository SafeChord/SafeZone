import os
from dotenv import load_dotenv

# load environment variables
load_dotenv()

# Service settings
SERVICE_NAME = os.getenv("SERVICE_NAME", "CovidDataIngestor")
SERVICE_VERSION = os.getenv("SERVICE_VERSION", "0.1.0")
# Server settings
SERVER_IP = os.getenv("SERVER_IP", "0.0.0.0")
SERVER_PORT = int(os.getenv("SERVER_PORT", "8000"))
# Logging settings
LOG_LEVEL = os.getenv("LOG_LEVEL", "DEBUG")
# Kafka settings
KAFKA_BOOTSTRAP = os.getenv("KAFKA_BOOTSTRAP", "localhost:9092")
KAFKA_TOPIC = os.getenv("KAFKA_TOPIC", "covid.case.data")
KAFKA_PUBLISH_TIMEOUT = float(os.getenv("KAFKA_PUBLISH_TIMEOUT", "4.0"))
