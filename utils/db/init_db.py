import json
from sqlalchemy import create_engine
from config.settings import DB_URL
from common.db.schema import cities, regions

# create the connection to the database
engine = create_engine(DB_URL)

geo_file_path = "common/geo_data/taiwan_geo_data.json"

with open(geo_file_path, encoding="utf-8") as f:
    geo_data = json.load(f)

with engine.connect() as connection:   
    for key in geo_data:
        result = connection.execute(cities.insert().values(
            {
                "name": key
            }
        )).returning(cities.c.id)
        
        cities_id = result.fetchone()[0]

        for value in geo_data[key]:
            connection.execute(regions.insert().values(
                {
                    "name": value,
                    "city_id": cities_id
                }
            ))