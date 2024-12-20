import json
import csv
from sqlalchemy import create_engine, select
from sqlalchemy.orm import Session
from config.settings import DB_URL
from utils.db.schema import cities, regions, populations, metadata


def init_db(engine=create_engine(DB_URL)):
    """
    Initialize the database with the taiwan geograph data and population data.

    """
    geo_file_path = "/app/utils/geo_data/taiwan_geo_data.json"

    with open(geo_file_path, encoding="utf-8") as f:
        geo_data = json.load(f)

    # Create all tables
    metadata.create_all(engine)

    # Use Session to interact with the database in SQLAlchemy 2.0 style
    with Session(engine) as session:
        for key in geo_data:
            # Insert city and get its ID
            insert_stmt = cities.insert().values(name=key)
            result = session.execute(insert_stmt)
            cities_id = result.inserted_primary_key[0]

            # Check if the city is created successfully
            city_query = select(cities).where(cities.c.id == cities_id)
            city_result = session.execute(city_query).fetchone()
            if city_result:
                print(f"city '{key}', is created, id {city_result}")
            else:
                print(f"city '{key}', is not created")

            # Insert regions for the city
            for value in geo_data[key]:
                print(f"city = {key}, region = {value}")
                session.execute(regions.insert().values(name=value, city_id=cities_id))

        # Commit the transaction
        session.commit()

    # init population data for each region
    population_file_path = "/app/utils/geo_data/towns_population.csv"

    with open(population_file_path, encoding="utf-8") as f:
        population_data = csv.DictReader(f)
        # Skip the header
        next(population_data , None)

        for row in population_data:
            city = row["COUNTY"]
            region = row["TOWN"]
            population = row["P_CNT"]

            with Session(engine) as session:
                # Get city_id and region_id
                city_id = session.execute(
                    select(cities.c.id).where(cities.c.name == city)
                ).fetchone()[0]
                region_id = session.execute(
                    select(regions.c.id).where(
                        regions.c.city_id == city_id,
                        regions.c.name == region)
                ).fetchone()[0]
                # Insert population data
                session.execute(
                    populations.insert().values(
                        city_id=city_id, region_id=region_id, population=population
                    )
                )
                print(f"city = {city}, region = {region}, population = {population}")
                session.commit()

init_db()
