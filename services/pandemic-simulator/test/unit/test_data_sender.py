# /test/unit_test/test_data_sender
import json
import logging

import respx # type: ignore
import pytest # type: ignore
from httpx import Response # type: ignore

from core.settings import INGESTOR_URL # type: ignore
from services.data_sender import send_data # type: ignore

@pytest.fixture(scope="module")
def logger():
    return logging.getLogger(__name__)

# import test cases
with open("/test/cases/test_data_sender.json", encoding="utf-8") as f:
    test_cases = json.load(f)


@respx.mock
@pytest.mark.asyncio
@pytest.mark.parametrize("case", test_cases, ids=lambda case: case["test_describes"]) # testing case by case
async def test_send_data(case, logger):
    # simulate request by requests_mock 
    mock_url = INGESTOR_URL + "/covid_event"
    respx.post(mock_url).mock(return_value=Response(200, json=case["expected_response"]))
    
    try:
        await send_data(case["data"])
        # if no exception is raised, the test is successful
        logger.debug("Data sending success.")
        assert True
    except Exception as e:  
        # if an exception is raised, the test is failed
        logger.debug(f"Data sending failed. {str(e)}")
        assert False, f"Data sending failed. {str(e)}"


@pytest.mark.asyncio
async def test_send_data_raises_service_validation_error():
    from exceptions.custom import ServiceValidationError

    invalid_data = [
        {"date": "2023-07-01", "city": "台北市", "region": "中山區", "cases": 0}
    ]
    with pytest.raises(ServiceValidationError) as exc_info:
        await send_data(invalid_data)

    errors = exc_info.value.errors
    assert any("cases" in err.get("loc", ()) for err in errors)


@respx.mock
@pytest.mark.asyncio
async def test_send_data_raises_http_status_error():
    import httpx

    mock_url = INGESTOR_URL + "/covid_event"
    respx.post(mock_url).mock(return_value=Response(500, json={"error": "failed"}))

    valid_data = [
        {"date": "2023-03-20", "city": "台北市", "region": "中山區", "cases": 1}
    ]
    with pytest.raises(httpx.HTTPStatusError):
        await send_data(valid_data)
