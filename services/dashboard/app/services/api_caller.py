import logging
import uuid
import requests

from utils.pydantic_model.request import NationalParameters, CityParameters, RegionParameters 
from utils.pydantic_model.response import AnalyticsAPIResponse

from config.settings import API_URL

logger = logging.getLogger(__name__)

class UnexceptedResponse(Exception):
    def __init__(self, response):
        message=f"Unexcepted response: {response}"
        super().__init__(message) 

def general_update(model, path):

    req_uuid = str(uuid.uuid4())

    logger.info("Sent request to analytics api.", extra={"trace_id": req_uuid})

    url = f"{API_URL}/{path}"
    response = requests.get(url, headers={"X-Trace-ID": req_uuid}, params=model.model_dump())

    # raise an error if the request was not successful by http status code
    response.raise_for_status()

    logger.info("Get response from analytics api.", extra={"trace_id": req_uuid})

    # check if the request was successful, it should a model of APIResponse
    api_response = AnalyticsAPIResponse(**response.json()).model_dump(exclude_none=True)
    # raise an error if the response status is not success
    if not api_response["success"]:
        raise UnexceptedResponse(api_response)
    # get the data from the response
    data = api_response["data"]
    return data

def update_region(now, city, region, interval, ratio):
    # create a RegionParameters model and validate the inputs
    model = RegionParameters(now=now, interval=interval, city=city, region=region, ratio=ratio)
    return general_update(model, "cases/region")

def update_city(now, city, interval, ratio):
    # create a CityParameters model and validate the inputs
    model = CityParameters(now=now, interval=interval, city=city, ratio=ratio)
    return general_update(model, "cases/city")

def update_national(now, interval):
    # create a NationalParameters model and validate the inputs
    model = NationalParameters(now=now, interval=interval)
    data = general_update(model, "cases/national")
    # check the content of the response data
    if data["end_date"] != now:
        raise UnexceptedResponse("The end date is unexpected")
    return data["aggregated_cases"]