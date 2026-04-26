# /test/unit_test/test_data_sender
import json
import logging

import respx # type: ignore
import pytest # type: ignore
from httpx import Response # type: ignore

from config.settings import INGESTOR_URL # type: ignore    
from pipeline.data_sender import send_data # type: ignore

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
