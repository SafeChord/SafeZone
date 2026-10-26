# tools/cli/relay/bin/system_helper.py
import requests # type: ignore
import redis # type: ignore
import psycopg2 # type: ignore

from config.settings import REDIS_HOST, REDIS_PORT, REDIS_PASSWORD, CACHE_HOST, CACHE_PORT, CACHE_PASSWORD
from config.settings import DB_URL, REPLICA_URL
from config.settings import SIMULATOR_URL, INGESTOR_URL, ANALYTICS_API_URL, DASHBOARD_URL, MKDOC_URL


def get_health(target: str="all"):
    status = {}
    all = (target == "all")
    if all or target == "cli-relay":
        status["cli-relay"] = "healthy"
    if all or target == "db":
        status["db"] = "healthy" if _check_db() else "unhealthy"
    if all or target == "redis-state":
        status["redis-state"] = "healthy" if _check_redis() else "unhealthy"
    if all or target == "redis-cache":
        status["redis-cache"] = "healthy" if _check_cache() else "unhealthy"
    if all or target == "simulator":
        status["simulator"] = "healthy" if _check_core(service="simulator") else "unhealthy"
    if all or target == "ingestor":
        status["ingestor"] = "healthy" if _check_core(service="ingestor") else "unhealthy"
    if all or target == "analytics-api":
        status["analytics-api"] = "healthy" if _check_core(service="analytics-api") else "unhealthy"
    if all or target == "dashboard":
        status["dashboard"] = "healthy" if _check_ui(service="dashboard") else "unhealthy"
    return status

def _check_db():
    try:
        conn = psycopg2.connect(DB_URL, connect_timeout=3)
        conn.close()

        conn = psycopg2.connect(REPLICA_URL, connect_timeout=3)
        conn.close()
    except psycopg2.OperationalError:
        return False
    return True
    
def _check_redis():
    try:
        r = redis.StrictRedis(
            host=REDIS_HOST,
            port=REDIS_PORT,
            password=REDIS_PASSWORD,
            decode_responses=True
        )
        if not r.ping():
            return False
    except redis.ConnectionError:
        return False
    return True

def _check_cache():
    try:
        r = redis.StrictRedis(
            host=CACHE_HOST,
            port=CACHE_PORT,
            password=CACHE_PASSWORD,
            decode_responses=True
        )
        if not r.ping():
            return False
    except redis.ConnectionError:
        return False
    return True

def _check_core(service):
    if service == "simulator":
        check_point = SIMULATOR_URL
    elif service == "ingestor":
        check_point = INGESTOR_URL
    elif service == "analytics-api":
        check_point = ANALYTICS_API_URL
    else:
        return False

    try:
        resp = requests.get(check_point + "/health")
        if resp.status_code != 200:
            return False
    except requests.exceptions.RequestException:
        return False
    return True

def _check_ui(service):
    if service == "dashboard":
        check_point = DASHBOARD_URL
    elif service == "mkdoc":
        check_point = MKDOC_URL
    else:
        return False
    try:
        resp = requests.get(check_point + "/health")
        if resp.status_code != 200:
            return False
    except requests.exceptions.RequestException:
        return False
    return True