"""Shared test fixtures for analytics-api.

Mirrors the DI layer that will live in api/dependencies.py (scaffold target).
All fixtures use container paths (/db, /data, /test) matching Dockerfile.test.
"""
import csv
import logging
from datetime import datetime

import pytest
from sqlalchemy import create_engine, select, and_
from sqlalchemy.orm import Session, sessionmaker

from utils.db.schema import cities, regions, covid_cases
from core.settings import DB_URL
from services.cache_service import get_city_region_cache, get_populations_cache

logger = logging.getLogger("conftest")


# ---------------------------------------------------------------------------
# Database fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def db_engine():
    """Session-scoped SQLAlchemy engine bound to test.db."""
    engine = create_engine(DB_URL)
    yield engine
    engine.dispose()


@pytest.fixture(scope="session")
def db_session(db_engine):
    """Session-scoped sessionmaker — matches app DI pattern."""
    return sessionmaker(bind=db_engine)


@pytest.fixture(scope="session", autouse=True)
def seed_business_data(db_engine):
    """One-time CSV -> covid_cases insert. Replaces duplicated init_db()."""
    with open("/data/test_data.csv", encoding="utf-8") as f:
        reader = csv.DictReader(f)

        with Session(db_engine) as session:
            for row in reader:
                city_name = row["city"]
                region_name = row["region"]

                city_result = session.execute(
                    select(cities).where(cities.c.name == city_name)
                ).fetchone()
                city_id = city_result.id

                region_result = session.execute(
                    select(regions).where(
                        and_(regions.c.city_id == city_id, regions.c.name == region_name)
                    )
                ).fetchone()
                region_id = region_result.id

                session.execute(
                    covid_cases.insert().values(
                        date=datetime.strptime(row["date"], "%Y-%m-%d"),
                        cases=row["cases"],
                        city_id=city_id,
                        region_id=region_id,
                    )
                )
            session.commit()

    logger.info("Test business data seeded successfully.")


# ---------------------------------------------------------------------------
# Cache fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(scope="session")
def geo_cache(db_session):
    """City-region mapping from DB — same function that will live in services/cache_service.py."""
    return get_city_region_cache(db_session)


@pytest.fixture(scope="session")
def populations_cache(db_session):
    """Population data from DB — same function that will live in services/cache_service.py."""
    return get_populations_cache(db_session)


# ---------------------------------------------------------------------------
# Integration fixtures
# ---------------------------------------------------------------------------

@pytest.fixture(scope="module")
def client(db_session, geo_cache, populations_cache):
    """FastAPI TestClient with dependency_overrides for test DB session.

    Uses app.dependency_overrides to inject test DB — no Redis needed.
    Will wire up to api/dependencies.py providers in Phase 2.
    """
    from fastapi.testclient import TestClient
    from main import create_app

    app = create_app()
    # dependency_overrides will be wired in Phase 2 when api/dependencies.py exists
    with TestClient(app) as c:
        yield c


@pytest.fixture(scope="module")
def mock_cache():
    """Mutable AsyncMock Redis cache client — tests set .get.return_value per call."""
    from unittest.mock import AsyncMock
    m = AsyncMock()
    m.get = AsyncMock(return_value=None)
    m.setex = AsyncMock()
    return m


@pytest.fixture(scope="module")
def client_with_cache(mock_cache):
    """TestClient with mock Redis injected via dependency_overrides.

    cache_client and cache_version are injected via Depends — no app.state needed.
    """
    from fastapi.testclient import TestClient
    from main import create_app
    from api.dependencies import get_cache_client, get_cache_version

    app = create_app()
    app.dependency_overrides[get_cache_client] = lambda: mock_cache
    app.dependency_overrides[get_cache_version] = lambda: "test-v1"
    with TestClient(app) as c:
        yield c
    app.dependency_overrides.clear()


@pytest.fixture(scope="module")
def client_broken():
    """TestClient with a DB session override that always raises.

    Verifies global_exception_handler catches unexpected errors and returns 500.
    Requires api/dependencies.py to exist — RED until Phase 2.
    """
    from fastapi.testclient import TestClient
    from main import create_app
    from api.dependencies import get_db_session  # noqa: RED until Phase 2

    def broken_db():
        raise RuntimeError("Simulated unexpected DB failure")

    app = create_app()
    app.dependency_overrides[get_db_session] = broken_db
    with TestClient(app, raise_server_exceptions=False) as c:
        yield c
    app.dependency_overrides.clear()
