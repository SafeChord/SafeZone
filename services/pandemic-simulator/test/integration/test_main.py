import json
import logging

import respx # type: ignore
import pytest # type: ignore
from httpx import Response # type: ignore
from fastapi.testclient import TestClient # type: ignore

from config.settings import INGESTOR_URL # type: ignore
from main import create_app # type: ignore

client = TestClient(create_app())

@pytest.fixture(scope="module")
def logger():
    return logging.getLogger(__name__)

# Import test cases
with open("/test/cases/test_integration.json", encoding="utf-8") as f:
    test_cases = json.load(f)

@respx.mock
@pytest.mark.parametrize("case", test_cases, ids=lambda case: case["test_describes"]) # testing case by case
def test_data_product(case, logger):

    endpoint = case["endpoint"]
    mock_url = INGESTOR_URL + "/covid_event"

    # Mock CovidDataIngestor behavior by returning a predefined response if 'expected_response' is present in the test case
    if "expected_response" in case:
        respx.post(mock_url).mock(return_value=Response(200, json=case["expected_response"]))
    else:
        respx.post(mock_url).mock(return_value=Response(200, json={}))

    # Handle daily requests
    if endpoint == "/simulate/daily":
        date = case.get("date", "")
        if not date:
            response = client.get("/simulate/daily")
        else:
            response = client.get(f"/simulate/daily?date={date}")
        assert response.status_code == case["expected_status_code"]

        # remove timestamp from response for comparison
        response_body = response.json()
        if "timestamp" in response_body:
            del response_body["timestamp"]
        # remove items if value is None
        response_body = {k: v for k, v in response_body.items() if v is not None}

        if "expected_response" in case:
            assert response_body == case["expected_response"]

    # Handle interval requests
    # Check if both start_date and end_date are provided in the test case
    elif endpoint == "/simulate/interval":
        start_date = case.get("start_date", "")
        end_date = case.get("end_date", "")
        if not (start_date and end_date):
            # If either start_date or end_date is missing, make a request without query parameters
            response = client.get("/simulate/interval")
        else:
            # If both start_date and end_date are provided, make a request with the query parameters
            response = client.get(
                f"/simulate/interval?start_date={start_date}&end_date={end_date}"
            )
        assert response.status_code == case["expected_status_code"]

        # remove timestamp from response for comparison
        response_body = response.json()
        if "timestamp" in response_body:
            del response_body["timestamp"]
        # remove items if value is None
        response_body = {k: v for k, v in response_body.items() if v is not None}
       
        if "expected_response" in case:
            assert response_body == case["expected_response"]

    # Handle endpoints that are not /simulate/daily or /simulate/interval
    else:
        response = client.get(endpoint)
        assert response.status_code == case["expected_status_code"]
