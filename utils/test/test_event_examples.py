# Every example event under contract/examples/ satisfies contract/covid_event.json.
# The worker's WK-R10 test relies on this: it checks that the worker accepts each example
# and does not validate them against the schema itself.
import glob
import json

import pytest  # type: ignore
from jsonschema import Draft202012Validator  # type: ignore

CONTRACT = "/contract/covid_event.json"
EXAMPLES = sorted(glob.glob("/contract/examples/covid_event/*.json"))

with open(CONTRACT, encoding="utf-8") as f:
    validator = Draft202012Validator(
        json.load(f), format_checker=Draft202012Validator.FORMAT_CHECKER
    )


def test_examples_exist():
    assert EXAMPLES, "no example events under contract/examples/covid_event/"


@pytest.mark.parametrize("path", EXAMPLES)
def test_example_satisfies_contract(path):
    with open(path, encoding="utf-8") as f:
        event = json.load(f)
    errors = [e.message for e in validator.iter_errors(event)]
    assert not errors, errors
