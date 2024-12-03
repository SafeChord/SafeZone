from sqlalchemy.orm import relationship, declarative_base
from utils.db.schema import cities, regions, covid_cases

Base = declarative_base()


class City(Base):
    __table__ = cities

    regions = relationship("Region", back_populates="city")
    cases = relationship("CovidCase", back_populates="city")


class Region(Base):
    __table__ = regions

    city = relationship("City", back_populates="regions")
    cases = relationship("CovidCase", back_populates="region")


class CovidCase(Base):
    __table__ = covid_cases

    region = relationship("Region", back_populates="cases")
    city = relationship("City", back_populates="cases")

    def to_dict(self):
        return {
            "id": self.id,
            "date": self.date.strftime("%Y-%m-%d") if self.date else None,
            "city_id": self.city_id,
            "region_id": self.region_id,
            "cases": self.cases,
        }
