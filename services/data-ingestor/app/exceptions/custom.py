from pydantic import ValidationError  # type: ignore


class KafkaUnavailableError(Exception):
    def __init__(self, message="Kafka producer is not available."):
        super().__init__(message)
