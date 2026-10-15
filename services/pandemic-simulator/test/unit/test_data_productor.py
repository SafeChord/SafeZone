# /test/unit_test/test_data_productor
import json
import logging

import pytest # type: ignore

from exceptions.custom import EmptyDataError # type: ignore
from services.data_productor import get_data_by_date, get_data_by_interval # type: ignore


@pytest.fixture(scope="module")
def logger():
    return logging.getLogger(__name__)


# import test cases
with open("/test/cases/test_data_productor.json", encoding="utf-8") as f:
    test_cases = json.load(f)

# testing case by case
@pytest.mark.parametrize("case", test_cases, ids=lambda case: case["test_describes"])
def test_data_product(case, logger):
    if case["component_under_test"] == "daily_data_productor":
        try:
            data = get_data_by_date(case["date"])
            assert data == case["expected_filter_data"]
            logger.debug("Data productor success.")
        except EmptyDataError:
            logger.debug("Data productor failed because of empty data.")
            assert True

    if case["component_under_test"] == "interval_data_productor":
        try:
            data = get_data_by_interval(case["start_date"], case["end_date"])
            assert data == case["expected_filter_data"]
            logger.debug("Data productor success.")
        except EmptyDataError:
            logger.debug("Data productor failed because of empty data.")
            assert True


def test_read_csv_cached_once(monkeypatch):
    import pandas as pd
    from services.data_productor import read_csv, get_data_by_date

    read_csv.cache_clear()
    call_count = 0
    orig_read_csv = pd.read_csv

    def counting_read_csv(*args, **kwargs):
        nonlocal call_count
        call_count += 1
        return orig_read_csv(*args, **kwargs)

    monkeypatch.setattr(pd, "read_csv", counting_read_csv)

    get_data_by_date("2023-03-20")
    assert call_count == 1

    get_data_by_date("2023-03-20")
    assert call_count == 1

    get_data_by_date("2023-07-01")
    assert call_count == 1
