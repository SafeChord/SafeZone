import json
import datetime
import random

import pytest # type: ignore
import responses # type: ignore
from jinja2 import Template # type: ignore
from unittest.mock import patch # type: ignore
from freezegun import freeze_time # type: ignore
from pydantic import ValidationError # type: ignore

from services.update_cases import get_region_data
from config.settings import API_URL


# define the fixture
# freeze the time to the target date
@pytest.fixture
def frozen_time():
    def _freeze_time(target_date):
        return freeze_time(target_date)

    return _freeze_time


def generate_mocks(case, interval, ratio=False):
    mock_data, mock_requests = {}, []
    # generate mock request
    url_template = case["url_template"]
    response_template = case["expected"]["response_template"]

    # get the start_date and end_date
    end_date = datetime.date.today()
    start_date = end_date - datetime.timedelta(days=int(interval))
    # format the date
    start_date = start_date.strftime("%Y-%m-%d")
    end_date= end_date.strftime("%Y-%m-%d")
    
    for mock_city, regions in case["mock_regions"].items():
        for mock_region in regions:
            data = 0
            # the context for the jinja2 template
            context = {
                "now": end_date,
                "start_date": start_date,
                "mock_city": mock_city,
                "mock_region": mock_region,
            }
            # simluated the different ratio feild
            if ratio:
                # it's should be generated before the context,
                # becasue the different render will cause the different result
                rand_ratio = round(random.uniform(0, 1), 5)
                # add the data to context
                context["cases_population_ratio"] = rand_ratio
                data = rand_ratio
            else:
                rand_cases = random.randint(0, 1000)
                context["aggregated_cases"] = rand_cases
                data = rand_cases

            # add the data to the mock_data
            mock_data[mock_region] = data
            
            # add other context
            context["ratio"] = ratio
            context["interval"] = interval
            context["API_URL"] = API_URL

            # render the url and response    
            url = Template(json.dumps(url_template)).render(context).strip('"')
            response = Template(json.dumps(response_template)).render(context)
            response = json.loads(response)

            mock_requests.append({"url": url, "json": response, "status_code": 200})
    return mock_requests, mock_data


# load the test cases
def load_test_cases(file_name):
    test_case_path = "/test/cases/test_services/get_region_data/"
    with open(test_case_path + file_name, encoding="utf-8") as f:
        return json.load(f)


# get the test case description
def get_case_describes(case):
    return case["test_describes"]


# test the update_cases function in correct scenarios
@patch("services.update_cases.load_taiwan_admin")
@pytest.mark.parametrize(
    "case", load_test_cases("cases_success.json"), ids=get_case_describes
)
@responses.activate
def test_update_trends(mock_load_geo, case, frozen_time):
    # mock the load_taiwan_admin function
    mock_load_geo.return_value = case["mock_regions"]
    mock_date = "2023-06-26"

    with frozen_time(mock_date):
        # get the parameters
        city = case["params"]["city"]
        interval = case["params"]["interval"]
        ratio = case["params"]["ratio"]
        
        # mock the requests
        mock_requests, mock_data = generate_mocks(case, interval, ratio)
        
        # simulate request by requests_mock
        for mock_request in mock_requests:
            url = mock_request["url"]
            mock_response = mock_request["json"]
            status_code = mock_request["status_code"]
            responses.add(responses.GET, url, json=mock_response, status=status_code)

        # call the function
        data = get_region_data(mock_date, city, interval, ratio)

        # assert the data's correctness
        assert data == mock_data


# test the update_cases function in parameters error scenarios
@patch("services.update_cases.load_taiwan_admin")
@pytest.mark.parametrize(
    "case", load_test_cases("cases_params_error.json"), ids=get_case_describes
)
def test_update_map_params_error(mock_load_geo, case, frozen_time):
    # mock the load_taiwan_admin function
    mock_load_geo.return_value = case["mock_regions"]
    mock_date = "2023-06-26"

    # get the parameters
    city = case["params"]["city"]
    interval = case["params"]["interval"]
    
    # expect the validation error beacuse the interval is not in the correct format
    with pytest.raises(ValidationError):
        get_region_data(mock_date, city, interval)


# test the update_cases function in response error scenarios
@patch("services.update_cases.load_taiwan_admin")
@pytest.mark.parametrize(
    "case", load_test_cases("cases_resp_error.json"), ids=get_case_describes
)
@responses.activate
def test_update_map_resp_error(mock_load_geo, case, frozen_time):
    # mock the load_taiwan_admin function
    mock_load_geo.return_value = case["mock_regions"]
    mock_date = "2023-06-26"

    with frozen_time(mock_date):
        # get the parameters
        city = case["params"]["city"]
        interval = case["params"]["interval"]
        ratio = case["params"]["ratio"]
        
        # mock the requests
        mock_requests, _ = generate_mocks(case, interval, ratio)
        
        # simulate request by requests_mock
        for mock_request in mock_requests:
            url = mock_request["url"]
            mock_response = mock_request["json"]
            status_code = mock_request["status_code"]
            responses.add(responses.GET, url, json=mock_response, status=status_code)
        
        # expect the validation error beacuse the response missing the end_date field 
        with pytest.raises(ValidationError):
            get_region_data(mock_date, city, interval)
