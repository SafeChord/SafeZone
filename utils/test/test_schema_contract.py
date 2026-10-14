# STD-R4, API-R9: utils/db/schema.py implements the table contract, contract/schema.sql.
# The Analytics API reads the tables through schema.py, so this test is how it honours
# the contract. Whitespace is ignored and `--` comment lines in schema.sql are dropped;
# otherwise schema.sql keeps the form SQLAlchemy emits (ADR-033).
from sqlalchemy.dialects import postgresql  # type: ignore
from sqlalchemy.schema import CreateTable  # type: ignore

from utils.db.schema import metadata

SCHEMA_SQL = "/contract/schema.sql"


def normalize(sql: str) -> str:
    lines = [line for line in sql.splitlines() if not line.lstrip().startswith("--")]
    return " ".join(" ".join(lines).split())


def generated_sql() -> str:
    dialect = postgresql.dialect()
    return "\n\n".join(
        str(CreateTable(table).compile(dialect=dialect)).strip() + ";"
        for table in metadata.sorted_tables
    )


def test_schema_py_matches_schema_sql():
    with open(SCHEMA_SQL, encoding="utf-8") as f:
        expected = f.read()
    assert normalize(generated_sql()) == normalize(expected), (
        "utils/db/schema.py differs from contract/schema.sql. SQL generated from schema.py:\n\n"
        + generated_sql()
    )
