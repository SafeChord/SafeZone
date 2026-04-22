import os
import sys
import csv
import logging
from datetime import datetime

# Add SafeZone root to sys.path
current_dir = os.path.dirname(os.path.abspath(__file__))
safezone_root = os.path.abspath(os.path.join(current_dir, "../../../.."))
sys.path.insert(0, safezone_root)

from sqlalchemy import create_engine, select, and_
from sqlalchemy.orm import Session
from utils.db.schema import cities, regions, covid_cases

logging.basicConfig(level=logging.INFO, format="%(asctime)s - %(levelname)s - %(message)s")
logger = logging.getLogger(__name__)

def seed_data():
    # Paths relative to this script
    db_path = os.path.abspath(os.path.join(current_dir, "../db/test_new.db"))
    csv_path = os.path.abspath(os.path.join(current_dir, "../data/test_data.csv"))

    if not os.path.exists(db_path):
        logger.error(f"Database not found at {db_path}. Please run generate-test-db.sh first.")
        sys.exit(1)

    db_url = f"sqlite:///{db_path}"
    engine = create_engine(db_url)

    logger.info(f"Seeding business data from {csv_path} to {db_path}...")

    with open(csv_path, encoding="utf-8") as f:
        reader = csv.DictReader(f)
        
        with Session(engine) as session:
            geo_id_map = {} # (city_name, region_name) -> (city_id, region_id)
            count = 0
            
            for row in reader:
                city_name = row["city"]
                region_name = row["region"]
                # Handle potential date parsing errors
                try:
                    case_date = datetime.strptime(row["date"], "%Y-%m-%d").date()
                except ValueError:
                    logger.error(f"Invalid date format: {row['date']}")
                    continue
                    
                cases_count = int(row["cases"])

                key = (city_name, region_name)
                if key not in geo_id_map:
                    city_res = session.execute(select(cities.c.id).where(cities.c.name == city_name)).fetchone()
                    if not city_res:
                        logger.warning(f"City '{city_name}' not found. Skipping.")
                        continue
                    city_id = city_res[0]

                    region_res = session.execute(
                        select(regions.c.id).where(
                            and_(regions.c.city_id == city_id, regions.c.name == region_name)
                        )
                    ).fetchone()
                    if not region_res:
                        logger.warning(f"Region '{region_name}' in '{city_name}' not found. Skipping.")
                        continue
                    region_id = region_res[0]
                    geo_id_map[key] = (city_id, region_id)
                else:
                    city_id, region_id = geo_id_map[key]

                session.execute(
                    covid_cases.insert().values(
                        date=case_date,
                        cases=cases_count,
                        city_id=city_id,
                        region_id=region_id
                    )
                )
                count += 1
            
            session.commit()
            logger.info(f"Successfully seeded {count} case records into test_new.db")

if __name__ == "__main__":
    seed_data()
