"""Shared test fixtures for pandemic-simulator."""
import logging

import pytest


@pytest.fixture(scope="module")
def logger():
    return logging.getLogger(__name__)
