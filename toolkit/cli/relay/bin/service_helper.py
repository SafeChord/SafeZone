# /tools/relay/bin/service_caller.py
import requests  # type: ignore
import redis  # type: ignore

from utils.logging.baselogger import trace_id_var
from config.settings import SIMULATOR_URL, ANALYTICS_API_URL
from config.settings import REDIS_HOST, REDIS_PORT, REDIS_DB, REDIS_PASSWORD


def simulate(date: str, end_date: str = None):
    trace_id = str(trace_id_var.get())

    if end_date is None:
        params = f"?date={date}"
        response = requests.get(
            SIMULATOR_URL + "/simulate/daily" + params, 
            headers={"X-Trace-ID": trace_id}
        )
    else:
        params = f"?start_date={date}&end_date={end_date}"
        response = requests.get(
            SIMULATOR_URL + "/simulate/interval" + params,
            headers={"X-Trace-ID": trace_id},
        )
    response.raise_for_status()

    # The CLI 'simulate' command is the only way to update COVID data in the database.
    # To ensure data consistency for the frontend, we reset the cache version to invalidate any outdated cached data.
    _reset_cache_version(trace_id)
    
    return response.json()


def verify(date: str, interval: int, city: str, region: str, ratio: bool = False):
    trace_id = str(trace_id_var.get())

    params = f"?now={date}&interval={interval}"
    if city and region:
        params += f"&city={city}&region={region}&ratio={ratio}"
        response = requests.get(
            ANALYTICS_API_URL + "/cases/region" + params,
            headers={"X-Trace-ID": trace_id},
        )
    elif city:
        params += f"&city={city}&ratio={ratio}"
        response = requests.get(
            ANALYTICS_API_URL + "/cases/city" + params,
            headers={"X-Trace-ID": trace_id},
        )
    else:
        response = requests.get(
            ANALYTICS_API_URL + "/cases/national" + params,
            headers={"X-Trace-ID": trace_id},
        )
    response.raise_for_status()  # Raise an error for bad responses
    return {
        "body": response.json(),
        "headers": {
            "X-Cache-Status": response.headers.get("X-Cache-Status", ""),
        },
    }


def _reset_cache_version(trace_id: str):
    redis_client = redis.Redis(
        host=REDIS_HOST,
        port=REDIS_PORT,
        db=REDIS_DB,
        password=REDIS_PASSWORD,
        decode_responses=True,
    )
    redis_client.set("current_cache_version", trace_id)
