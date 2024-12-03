import json
from sqlalchemy import create_engine, select
from sqlalchemy.orm import Session
from config.settings import DB_URL
from utils.db.schema import cities, regions, metadata

def init_db(engine=create_engine(DB_URL)):
    """
    Initialize the database with the taiwan geograph data.
    
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
init_db()