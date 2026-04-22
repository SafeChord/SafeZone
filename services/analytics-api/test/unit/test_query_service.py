"""Unit tests for query_service — scaffold-compliant.

Rules:
- ZERO FastAPI imports — calls services.query_service.query_cases() directly
- Uses conftest fixtures (db_session, geo_cache, populations_cache)
- Parametrized via test/cases/test_query_service.json
- Import path targets NEW scaffold structure (will be RED until Phase 2)
"""
import json
from datetime import datetime

import pytest

from services.query_service import query_cases
from exceptions.custom import InvalidTaiwanRegionException, InvalidTaiwanCityException


with open("/test/cases/test_query_service.json", encoding="utf-8") as f:
    test_cases = json.load(f)


def get_case_describes(case):
    return case["test_describes"]


@pytest.mark.parametrize("case", test_cases, ids=get_case_describes)
def test_query_cases(case, db_session, geo_cache, populations_cache):
    params = dict(case["params"])
    params["start_date"] = datetime.strptime(params["start_date"], "%Y-%m-%d").date()
    params["end_date"] = datetime.strptime(params["end_date"], "%Y-%m-%d").date()

    try:
        result = query_cases(db_session, geo_cache, populations_cache, params)
        assert result == case["expected"]["query_result"]
    except InvalidTaiwanRegionException as exc:
        assert case["expected"]["exception"] == "InvalidTaiwanRegionException"
        assert str(exc) == case["expected"]["exception_detail"]
    except InvalidTaiwanCityException as exc:
        assert case["expected"]["exception"] == "InvalidTaiwanCityException"
        assert str(exc) == case["expected"]["exception_detail"]
