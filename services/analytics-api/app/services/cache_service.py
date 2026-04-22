# app/services/cache_service.py
import logging

from sqlalchemy import select  # type: ignore

from utils.db.orm import City, Region
from utils.db.schema import populations


logger = logging.getLogger(__name__)


# geo_cache stores city-region mapping in the format:
# city -> (city_id, {region_name -> region_id})
def get_city_region_cache(Session) -> dict[str, tuple[int, dict[str, int]]]:
    geo_cache = {}
    with Session() as session:
        results = (
            session.query(City, Region)
            .outerjoin(Region, Region.city_id == City.id)
            .all()
        )
        for city, region in results:
            if city.name not in geo_cache:
                geo_cache[city.name] = (city.id, {})
            if region:
                geo_cache[city.name][1][region.name] = region.id
    logger.debug("City-region cache loaded successfully.")
    return geo_cache


# populations_cache stores population data in the format:
# city_id -> {region_id -> population}
def get_populations_cache(Session) -> dict[int, dict[int, int]]:
    populations_cache = {}
    with Session() as session:
        populations_cache.clear()
        select_stmt = select(
            populations.c.city_id, populations.c.region_id, populations.c.population
        )
        for city_id, region_id, population in session.execute(select_stmt):
            if city_id not in populations_cache:
                populations_cache[city_id] = {}
            populations_cache[city_id][region_id] = population
    logger.debug("population cache loaded successfully.")
    return populations_cache
