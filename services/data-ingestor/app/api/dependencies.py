from fastapi import Request  # type: ignore


def get_kafka_producer(request: Request):
    """Provide the Kafka producer from app state."""
    return request.app.state.kafka_producer
