import json
from datetime import datetime
from common.custom_exceptions.exceptions import InvalidTaiwanCityException
from common.custom_exceptions.exceptions import InvalidTaiwanRegionException
from common.custom_exceptions.exceptions import InvalidCasesNumberException
from common.custom_exceptions.exceptions import InvalidDateFormatException

    
def validate_data(date, city, region, cases):
    """
    Validate the data to ensure it is in the correct format.

    This method validates the date, city, region, and cases fields of the CovidCase instance.
    """
    # Check the values in every field in the data
    validate_date(date)

    validate_geo(city, region)

    validate_cases(cases)
       
   
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
    with open("/app/common/geo_data/taiwan_geo_data.json", encoding="utf-8") as f:
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
