# app/api/dependencies.py
from fastapi import Request  # type: ignore
from sqlalchemy.orm import sessionmaker  # type: ignore


def get_db_session(request: Request) -> sessionmaker:
    """Provide the DB sessionmaker from app state."""
    return request.app.state.Session


def get_geo_cache(request: Request) -> dict:
    """Provide the city-region mapping cache from app state."""
    return request.app.state.city_region_cache


def get_populations_cache(request: Request) -> dict:
    """Provide the population data cache from app state."""
    return request.app.state.populations_cache


def get_cache_client(request: Request):
    """Provide the Redis cache client from app state. Returns None when unavailable."""
    return request.app.state.cache_client


def get_cache_version(request: Request) -> str:
    """Provide the current cache version from app state."""
    return request.app.state.cache_version
