"""Unit tests for core/lifecycle.py — generate_cache_key determinism.

No FastAPI context required: tests call the function directly.
"""
from datetime import date

from core.lifecycle import generate_cache_key
from utils.pydantic_model.request import NationalParameters, RegionParameters


def test_same_inputs_produce_same_key():
    params = NationalParameters(now=date(2023, 3, 26), interval="7")
    assert generate_cache_key("v1", "cases_national", params) == \
           generate_cache_key("v1", "cases_national", params)


def test_different_params_produce_different_keys():
    p1 = NationalParameters(now=date(2023, 3, 26), interval="7")
    p2 = NationalParameters(now=date(2023, 3, 27), interval="7")
    assert generate_cache_key("v1", "cases_national", p1) != \
           generate_cache_key("v1", "cases_national", p2)


def test_different_versions_produce_different_keys():
    params = NationalParameters(now=date(2023, 3, 26), interval="7")
    assert generate_cache_key("v1", "cases_national", params) != \
           generate_cache_key("v2", "cases_national", params)


def test_different_endpoints_produce_different_keys():
    p_national = NationalParameters(now=date(2023, 3, 26), interval="7")
    p_region = RegionParameters(now=date(2023, 3, 26), interval="7", city="台北市", region="信義區")
    assert generate_cache_key("v1", "cases_national", p_national) != \
           generate_cache_key("v1", "cases_region", p_region)


def test_key_format_version_endpoint_hash():
    params = NationalParameters(now=date(2023, 3, 26), interval="7")
    key = generate_cache_key("v1", "cases_national", params)
    version, endpoint, digest = key.split(":")
    assert version == "v1"
    assert endpoint == "cases_national"
    assert len(digest) == 64  # SHA256 hex digest
