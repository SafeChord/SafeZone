# exception_handler.py
from fastapi import HTTPException
from fastapi.responses import JSONResponse
from utils.custom_exceptions.exceptions import (
    InvalidTaiwanCityException,
    InvalidTaiwanRegionException,
    InvalidCasesNumberException,
)
from utils.custom_exceptions.exceptions import (
    MissingDataException,
    InvalidDateFormatException,
    ExtraFieldException,
    DataDuplicateException,
    InvalidContentTypeException
)
from utils.custom_exceptions.exceptions import InvalidDateRangeException, EmptyDataException
from requests.exceptions import ConnectionError, Timeout, HTTPError, RequestException


# handle exceptions and return the status code and message to api caller
def handle_exceptions(e):
    covid_case_exception_handler(e)
    simulator_exception_handler(e)
    request_exception_handler(e)
    raise HTTPException(status_code=400, detail=str(e))


def request_exception_handler(e):
    if isinstance(e, ConnectionError):
        raise HTTPException(status_code=503, detail="Failed to connect to the server.")
    elif isinstance(e, Timeout):
        raise HTTPException(status_code=504, detail="The request timed out.")
    elif isinstance(e, HTTPError):
        raise HTTPException(status_code=400, detail=f"HTTP error occurred: {str(e)}")
    elif isinstance(e, RequestException):
        raise HTTPException(status_code=500, detail=f"An unexpected error occurred: {str(e)}")


def covid_case_exception_handler(e):
    if isinstance(e, InvalidTaiwanCityException):
        raise HTTPException(status_code=422, detail=str(e))
    elif isinstance(e, InvalidTaiwanRegionException):
        raise HTTPException(status_code=422, detail=str(e))
    elif isinstance(e, InvalidCasesNumberException):
        raise HTTPException(status_code=422, detail=str(e))
    elif isinstance(e, MissingDataException):
        raise HTTPException(status_code=400, detail=str(e))
    elif isinstance(e, InvalidDateFormatException):
        raise HTTPException(status_code=422, detail=str(e))
    elif isinstance(e, ExtraFieldException):
        raise HTTPException(status_code=400, detail=str(e))
    elif isinstance(e, DataDuplicateException):
        raise HTTPException(status_code=409, detail=str(e))
    elif isinstance(e, InvalidContentTypeException):
        raise HTTPException(status_code=415, detail=str(e))


def simulator_exception_handler(e):
    if isinstance(e, InvalidDateRangeException):
        raise HTTPException(status_code=400, detail=str(e))
    elif isinstance(e, EmptyDataException):
        raise HTTPException(status_code=400, detail=str(e))
