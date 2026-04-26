"""Shared test fixtures for data-ingestor."""
from unittest.mock import AsyncMock, patch

import pytest
from fastapi.testclient import TestClient

from api.dependencies import get_kafka_producer
from main import create_app


@pytest.fixture(scope="module")
def mock_kafka_producer():
    """AsyncMock Kafka producer — tests can inspect send_and_wait calls."""
    producer = AsyncMock()
    producer.send_and_wait = AsyncMock()
    return producer


@pytest.fixture(scope="module")
def client(mock_kafka_producer):
    """FastAPI TestClient with mock Kafka producer injected via dependency_overrides.

    Patches lifespan startup/shutdown to avoid real Kafka connection.
    """
    with patch("main.startup_kafka", new_callable=AsyncMock, return_value=mock_kafka_producer):
        with patch("main.shutdown_kafka", new_callable=AsyncMock):
            app = create_app()
            app.dependency_overrides[get_kafka_producer] = lambda: mock_kafka_producer
            with TestClient(app) as c:
                yield c
            app.dependency_overrides.clear()
