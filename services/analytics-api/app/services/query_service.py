# app/services/query_service.py
import logging

from sqlalchemy import select, func  # type: ignore

from utils.db.schema import covid_cases
from exceptions.custom import InvalidTaiwanCityException, InvalidTaiwanRegionException


logger = logging.getLogger(__name__)

RATIO_FACTOR = 10000  # used to calculate cases to population ratio


def query_cases(db_session, geo_cache, populations_cache, params):
    if "city" in params and "region" in params:
        result = query_cases_by_region(db_session, geo_cache, populations_cache, params)
    elif "city" in params:
        result = query_cases_by_city(db_session, geo_cache, populations_cache, params)
    else:
        result = query_cases_national(db_session, params)
    return result


def query_cases_by_region(Session, geo_cache, populations_cache, params):
    logger.info(
        f"Query region cases with {params}.", extra={"event": "query_region_cases"}
    )

    with Session() as session:
        # check if city and region are valid
        if params["city"] not in geo_cache:
            raise InvalidTaiwanCityException(params["city"])
        if params["region"] not in geo_cache[params["city"]][1]:
            raise InvalidTaiwanRegionException(params["city"], params["region"])

        city_id = geo_cache[params["city"]][0]
        region_id = geo_cache[params["city"]][1][params["region"]]

        # query cases by region
        query = select(func.sum(covid_cases.c.cases).label("total_cases")).where(
            covid_cases.c.date >= params["start_date"],
            covid_cases.c.date <= params["end_date"],
            covid_cases.c.city_id == city_id,
            covid_cases.c.region_id == region_id,
        )
        cases = session.execute(query).scalar()
        cases = cases if cases else 0

    logger.info(
        f"Query region cases successful with {params}.",
        extra={"event": "query_region_cases_success"},
    )

    if "ratio" in params and params["ratio"]:
        population = populations_cache[city_id][region_id]
        cases = round(cases * RATIO_FACTOR / population, 5)

        return {
            "start_date": params["start_date"].strftime("%Y-%m-%d"),
            "end_date": params["end_date"].strftime("%Y-%m-%d"),
            "city": params["city"],
            "region": params["region"],
            "cases_population_ratio": cases,
        }
    else:
        return {
            "start_date": params["start_date"].strftime("%Y-%m-%d"),
            "end_date": params["end_date"].strftime("%Y-%m-%d"),
            "city": params["city"],
            "region": params["region"],
            "aggregated_cases": cases,
        }


def query_cases_by_city(Session, geo_cache, populations_cache, params):
    logger.info(f"Query city case with {params}.", extra={"event": "query_city_cases"})

    with Session() as session:
        # check if city is valid
        if params["city"] not in geo_cache:
            raise InvalidTaiwanCityException(params["city"])

        city_id = geo_cache[params["city"]][0]
        # query cases by city
        query = select(func.sum(covid_cases.c.cases).label("total_cases")).where(
            covid_cases.c.date >= params["start_date"],
            covid_cases.c.date <= params["end_date"],
            covid_cases.c.city_id == city_id,
        )
        cases = session.execute(query).scalar()
        cases = cases if cases else 0

    logger.info(
        f"Query city cases successful with {params}.",
        extra={"event": "query_city_cases_success"},
    )

    if "ratio" in params and params["ratio"]:
        city_population = 0
        for _, population in populations_cache[city_id].items():
            city_population += population
        cases = (
            round(cases * RATIO_FACTOR / city_population, 5)
            if city_population > 0
            else 0
        )

        return {
            "start_date": params["start_date"].strftime("%Y-%m-%d"),
            "end_date": params["end_date"].strftime("%Y-%m-%d"),
            "city": params["city"],
            "cases_population_ratio": cases,
        }

    else:
        return {
            "start_date": params["start_date"].strftime("%Y-%m-%d"),
            "end_date": params["end_date"].strftime("%Y-%m-%d"),
            "city": params["city"],
            "aggregated_cases": cases,
        }


def query_cases_national(Session, params):
    logger.info(
        f"Querying national cases with {params}.",
        extra={"event": "query_national_cases"},
    )

    with Session() as session:
        query = select(func.sum(covid_cases.c.cases).label("total_cases")).where(
            covid_cases.c.date >= params["start_date"],
            covid_cases.c.date <= params["end_date"],
        )
        cases = session.execute(query).scalar()
        cases = cases if cases else 0

    logger.info(
        f"Query national cases successful with {params}.",
        extra={"event": "query_national_cases_success"},
    )

    return {
        "start_date": params["start_date"].strftime("%Y-%m-%d"),
        "end_date": params["end_date"].strftime("%Y-%m-%d"),
        "aggregated_cases": cases,
    }
