from sqlalchemy import MetaData, Table, Column, ForeignKey, Date, Integer, String

metadata = MetaData()

# Cities Table
cities = Table(
    "cities",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("name", String(50), unique=True, nullable=False),
)

# Regions Table
regions = Table(
    "regions",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("name", String(50), nullable=False),
    Column("city_id", Integer, ForeignKey("cities.id"), nullable=False),
)

# Covid Cases Table
covid_cases = Table(
    "covid_cases",
    metadata,
    Column("id", Integer, primary_key=True),
    Column("date", Date, nullable=False),
    Column("city_id", Integer, ForeignKey("cities.id"), nullable=False),
    Column("region_id", Integer, ForeignKey("regions.id"), nullable=False),
    Column("cases", Integer, nullable=False),
)
