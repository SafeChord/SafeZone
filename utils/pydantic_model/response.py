from datetime import datetime, timezone
from typing import Optional

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
    up_date: str = Field(
        ..., description="The up time of the time server in 'YYYY-MM-DD' format."
    )
    # the date of whole system(if mock is enabled, it will be the mock date else it will be the current date)
    system_date: str = Field(..., description="The system date in 'YYYY-MM-DD' format.")
    accelerate: int = Field(
        ..., ge=1, le=10, description="The acceleration factor of the time server."
    )


# --- response model --- #
# base response model
class ErrorModel(BaseModel):
    fields: Optional[str] = Field(
        None, description="Short message describing the status."
    )
    detail: str = Field(..., description="Detailed explanation of the message.")


class APIResponse(BaseModel):
    success: bool = Field(..., description="State of the response.")
    message: str = Field(..., description="Short message describing the status.")
    errors: Optional[ErrorModel] = Field(None, description="Error detail.")
    # timestamp automatically generated with current UTC time in ISO 8601 format
    timestamp: str = Field(
        default_factory=lambda: datetime.now(timezone.utc)
        .isoformat()
        .replace("+00:00", "Z"),
        description="UTC timestamp of response generation",
    )


# analytics api response model
class AnalyticsAPIResponse(APIResponse):
    data: AnalyticsAPIData = Field(..., description="AnalyticsAPIData details.")


# time server response model
class MocktimeStatusResponse(APIResponse):
    data: MocktimeStatusData = Field(..., description="MocktimeStatusData details.")
