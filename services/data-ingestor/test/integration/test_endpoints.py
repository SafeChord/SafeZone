import json
import logging

import pytest  # type: ignore

@pytest.fixture(scope="module")
def logger():
    return logging.getLogger(__name__)


# Import test cases
with open("/test/cases/test_integration.json", encoding="utf-8") as f:
    test_cases = json.load(f)


def resp_filter(resp):
    # Filter the response to remove timestamp
    if "timestamp" in resp:
        del resp["timestamp"]
    # Filter the response to remove None value fields
    resp = {k: v for k, v in resp.items() if v is not None}
    return resp


@pytest.mark.parametrize(
    "case", test_cases, ids=lambda case: case["test_describes"]
)
def test_data_product(case, client, logger):
    endpoint = case["endpoint"]
    method = case["method"]

    if method == "GET":
        logger.debug(f"Testing GET request to {endpoint}")
        resp = client.get(endpoint)
    elif method == "POST":
        logger.debug(
            f"Testing POST request to {endpoint} with payload: {case['request']['payload']}"
        )
        resp = client.post(endpoint, json=case["request"]["payload"])

    if case["expected_status_code"] != 200:
        logger.debug(
            f"Expected error status code: {case['expected_status_code']}, got: {resp.status_code}"
        )
        assert resp.status_code == case["expected_status_code"]
    else:
        logger.debug(f"Expected success case, got status code: {resp.status_code}")
        assert resp.status_code == 200
        assert resp_filter(resp.json()) == case["expected_response"]
