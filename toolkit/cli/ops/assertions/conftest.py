import pytest

from bin.client import DataflowClient, TimeClient, HealthClient


@pytest.fixture(scope="session")
def dataflow_client():
    return DataflowClient()


@pytest.fixture(scope="session")
def time_client():
    return TimeClient()


@pytest.fixture(scope="session")
def health_client():
    return HealthClient()
