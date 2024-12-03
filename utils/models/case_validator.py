import json
from datetime import datetime
from utils.custom_exceptions.exceptions import InvalidTaiwanCityException
from utils.custom_exceptions.exceptions import InvalidTaiwanRegionException
from utils.custom_exceptions.exceptions import InvalidCasesNumberException
from utils.custom_exceptions.exceptions import InvalidDateFormatException
from utils.custom_exceptions.exceptions import ExtraFieldException
from utils.custom_exceptions.exceptions import MissingDataException
from config.logger import get_logger

logger = get_logger()
    
def validate_data(**kwargs):
    """
    Validate the data to ensure it is in the correct format.

    This method validates the date, city, region, and cases fields of the CovidCase instance.
    """
    excepted_fields = ["date", "city", "region", "cases"]
    not_expected_fields = []
    for field_name in kwargs.keys():
        # field name not in the expected fields raise the ExtraFieldException
        # in case: filed_name are duplicated in the request body it will raise the ExtraFieldException
        if field_name not in excepted_fields:
            not_expected_fields.append(field_name)
        # field name same as the one of the expected fields drop the elemnt from the excepted_fields
        if field_name in excepted_fields:
            excepted_fields.remove(field_name)
    
    # if not_expected_fields is not empty means some of the fields are not expected
    if len(not_expected_fields) > 0:
        raise ExtraFieldException(not_expected_fields)
    # if excepted_fields is not empty means some of the expected fields are missing 
    if len(excepted_fields) > 0:
        raise MissingDataException(excepted_fields)
    
    # Check the values in every field in the data
    logger.info(f"validating data = {kwargs["date"]}, {kwargs["city"]}, {kwargs["region"]}, {kwargs["cases"]}")
    validate_date(kwargs["date"])

    validate_geo(kwargs["city"], kwargs["region"])

    validate_cases(kwargs["cases"])
       
   
def validate_date(date):
    if not isinstance(date, str):
        raise InvalidDateFormatException
    try:
        datetime.strptime(date, "%Y-%m-%d")
    except ValueError:
        raise InvalidDateFormatException

   
def validate_geo(city, region):
    if not isinstance(city, str):
        raise InvalidTaiwanCityException
    if not isinstance(region, str):
        raise InvalidTaiwanRegionException

        # Load geo data once
    with open("/app/utils/geo_data/taiwan_geo_data.json", encoding="utf-8") as f:
        geo_data = json.load(f)

    if city not in geo_data:
        raise InvalidTaiwanCityException
    if region not in geo_data[city]:
        raise InvalidTaiwanRegionException

   
def validate_cases(cases):
    if not isinstance(cases, int):
        raise InvalidCasesNumberException
    if cases <= 0:
        raise InvalidCasesNumberException
