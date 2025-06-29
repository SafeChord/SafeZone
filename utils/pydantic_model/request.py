import datetime
from typing import Optional, Literal

from pydantic import BaseModel, Field, model_validator  # type: ignore


# --- payload/parameter models  --- #
# cli relay models
## endpoint /dataflow/simulate
class SimulateModel(BaseModel):
    date: datetime.date = Field(
        ..., description="Invalid date format. Expected 'YYYY-MM-DD'."
    )
    end_date: Optional[datetime.date] = Field(
        None, description="Invalid date format. Expected 'YYYY-MM-DD'."
    )


## endpoint /dataflow/verify
class VerifyModel(BaseModel):
    date: datetime.date = Field(
        ..., description="Invalid date format. Expected 'YYYY-MM-DD'."
    )
    interval: int = Field(1, description="Invalid interval format. Expected integer.")
    city: Optional[str] = Field(
        None, description="The length of city name is between 1 to 50."
    )
    region: Optional[str] = Field(
        None, description="The length of region name is between 1 to 50."
    )
    ratio: bool = Field(False, description="The data is ratio of population or not.")


## endpoint /health
class HealthCheckModel(BaseModel):
    target: Optional[str] = Field(None, description="Component (db, redis, mkdoc)")
    all: bool = Field(False, description="Check all components")


## endpoint /time/set，use SetTimeModel instead


# analytics api models
## endpoint /cases/national
class NationalParameters(BaseModel):
    now: datetime.date = Field(..., description="Date in 'YYYY-MM-DD' format.")
    interval: Literal["1", "3", "7", "14", "30"] = Field(
        ..., description="the aggergate data form number of days before now"
    )


## endpoint /cases/city
class CityParameters(NationalParameters):
    city: str = Field(
        ...,
        min_length=1,
        max_length=50,
        description="The length of city name is between 1 to 50.",
    )
    ratio: bool = Field(
        False, description="Whether to return the ratio of cases to population."
    )


## endpoint /cases/region
class RegionParameters(CityParameters):
    region: str = Field(
        ...,
        min_length=1,
        max_length=50,
        description="The length of region name is between 1 to 50.",
    )


# simulator models
## endpoint /simulate/interval
class IntervalParameters(BaseModel):
    start_date: datetime.date = Field(
        ..., description="Start date in 'YYYY-MM-DD' format."
    )
    end_date: datetime.date = Field(..., description="End date in 'YYYY-MM-DD' format.")

    @model_validator(mode="after")
    def validate_dates(cls, values):
        start_date = values.start_date
        end_date = values.end_date
        if end_date < start_date:
            raise ValueError("End date cannot be earlier than start date.")
        return values


## endpoint /simulate/daily
class DailyParameters(BaseModel):
    date: datetime.date = Field(..., description="Date in 'YYYY-MM-DD' format.")


# ingestor models
## endpoint /collect
class CovidDataModel(BaseModel):
    date: str = Field(
        ..., description="Invalid date format. Expected 'YYYY-MM-DD'."
    )
    city: str = Field(
        ...,
        min_length=1,
        max_length=50,
        description="The length of city name is between 1 to 50.",
    )
    region: str = Field(
        ...,
        min_length=1,
        max_length=50,
        description="The length of region name is between 1 to 50.",
    )
    cases: int = Field(..., ge=1, description="'cases' must be a positive integer.")


# time server models
## endpoint /time/set
class SetTimeModel(BaseModel):
    mock: bool = Field(False, description="Indicates whether to enable mock time.")
    mock_date: Optional[str] = Field(
        None, description="Invalid date format. Expected 'YYYY-MM-DD'."
    )
    acceleration: Optional[int] = Field(
        None, ge=1, le=10, description="Invalid acceleration format. Expected integer."
    )
    # 0: no acceleration, 1: 1x, 2: 2x, 3: 3x, etc.
