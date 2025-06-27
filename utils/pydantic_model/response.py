from datetime import datetime, timezone
from typing import Optional, List, Union

from pydantic import BaseModel, Field  # type: ignore


# --- data models  --- #
# analytics api models
class AnalyticsAPIData(BaseModel):
    start_date: str = Field(
        ..., description="Invalid date format. Expected 'YYYY-MM-DD'."
    )
    end_date: str = Field(
        ..., description="Invalid date format. Expected 'YYYY-MM-DD'."
    )
    city: Optional[str] = Field(
        None,
        min_length=1,
        max_length=50,
        description="The length of city name is between 1 to 50.",
    )
    region: Optional[str] = Field(
        None,
        min_length=1,
        max_length=50,
        description="The length of region name is between 1 to 50.",
    )
    aggregated_cases: Optional[int] = Field(
        None, ge=0, description="'cases' must be a positive integer."
    )
    cases_population_ratio: Optional[float] = Field(
        None,
        ge=0,
        le=100,
        description="'cases_population_ratio' must be a float between 0 and 1.",
    )


# time server models
class MocktimeStatusData(BaseModel):
    mock: bool = Field(
        ..., description="Indicates whether the time server is in mock mode."
    )
    mock_date: str = Field(..., description="The mock time in 'YYYY-MM-DD' format.")
    mock_update_time: str = Field(
        ...,
        description="The last update time of the mock date in 'YYYY-MM-DDTHH:MM:SS' format.",
    )
    launch_time: str = Field(
        ...,
        description="The up time of the time server in 'YYYY-MM-DDTHH:MM:SS' format.",
    )
    acceleration: int = Field(
        ..., ge=1, le=10, description="The acceleration factor of the time server."
    )
    # the date of whole system(if mock is enabled, it will be the mock date else it will be the current date)
    system_date: str = Field(..., description="The system date in 'YYYY-MM-DD' format.")


class ErrorModel(BaseModel):
    field: Optional[str] = Field(
        None, description="Field name that caused the error, or None if not applicable."
    )
    summary: Optional[str] = Field(
        None, description="Short summary of the error (e.g., 'Invalid value')."
    )
    detail: Optional[str] = Field(
        ..., description="Detailed explanation of the error (for debugging or logs)."
    )


# --- response model --- #
# base response model
class APIResponse(BaseModel):
    success: bool = Field(..., description="Indicates if the request was successful.")
    message: str = Field(..., description="Primary summary message for the response.")
    detail: Optional[str] = Field(None, description="Primary summary message for the response.")
    errors: Optional[Union[ErrorModel, List[ErrorModel]]] = Field(
        None, description="Error details. Can be a single error or a list of errors."
    )
    timestamp: str = Field(
        default_factory=lambda: datetime.now(timezone.utc)
        .isoformat()
        .replace("+00:00", "Z"),
        description="UTC timestamp (ISO8601) of when the response was generated.",
    )

class HealthResponse(APIResponse):
    status: str = Field(
        "healthy", description="Health status of the service, typically 'healthy'."
    )
    detail: Optional[str] = Field(
        None, description= "Additional details about the health status, if any."
    )


# analytics api response model
class AnalyticsAPIResponse(APIResponse):
    data: Optional[AnalyticsAPIData] = Field(None, description="AnalyticsAPIData details.")

# time server response model
class SystemDateResponse(APIResponse):
    system_date: Optional[str] = Field(
        None, description="The system date in 'YYYY-MM-DD' format."
    )

class MocktimeStatusResponse(APIResponse):
    data: Optional[MocktimeStatusData] = Field(None, description="MocktimeStatusData details.")
