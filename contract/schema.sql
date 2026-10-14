-- Table contract of SafeZone: the definition of the tables.
-- utils/db/schema.py is its Python implementation; a change is made here first, then in
-- schema.py until `make test-contract` passes. Keep the form SQLAlchemy emits for the
-- PostgreSQL dialect (ADR-033); whitespace and comment lines are ignored by the test.

CREATE TABLE cities (
	id SERIAL NOT NULL,
	name VARCHAR(50) NOT NULL,
	PRIMARY KEY (id),
	UNIQUE (name)
);

CREATE TABLE regions (
	id SERIAL NOT NULL,
	name VARCHAR(50) NOT NULL,
	city_id INTEGER NOT NULL,
	PRIMARY KEY (id),
	FOREIGN KEY(city_id) REFERENCES cities (id)
);

CREATE TABLE covid_cases (
	id SERIAL NOT NULL,
	date DATE NOT NULL,
	city_id INTEGER NOT NULL,
	region_id INTEGER NOT NULL,
	cases INTEGER NOT NULL,
	PRIMARY KEY (id),
	CONSTRAINT uix_date_city_region UNIQUE (date, city_id, region_id),
	FOREIGN KEY(city_id) REFERENCES cities (id),
	FOREIGN KEY(region_id) REFERENCES regions (id)
);

CREATE TABLE populations (
	id SERIAL NOT NULL,
	city_id INTEGER NOT NULL,
	region_id INTEGER NOT NULL,
	population INTEGER NOT NULL,
	PRIMARY KEY (id),
	CONSTRAINT uix_city_region UNIQUE (city_id, region_id),
	FOREIGN KEY(city_id) REFERENCES cities (id),
	FOREIGN KEY(region_id) REFERENCES regions (id)
);
