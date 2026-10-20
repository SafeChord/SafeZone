package postgres

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Cache struct {
	cityMap   map[string]int
	regionMap map[string]int
}

func NewCache(db *sqlx.DB) (*Cache, error) {
	cityMap, err := loadCityIDs(db)
	if err != nil {
		return nil, err
	}
	regionMap, err := loadRegionIDs(db)
	if err != nil {
		return nil, err
	}
	return &Cache{cityMap: cityMap, regionMap: regionMap}, nil
}

func loadCityIDs(db *sqlx.DB) (cityMap map[string]int, err error) {
	rows, err := db.Queryx("SELECT id, name FROM cities")
	if err != nil {
		return nil, fmt.Errorf("query cities: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close cities rows: %w", closeErr)
		}
	}()

	cityMap = make(map[string]int)
	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return nil, fmt.Errorf("scan city row: %w", err)
		}
		cityMap[name] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate cities rows: %w", err)
	}
	return cityMap, nil
}

func loadRegionIDs(db *sqlx.DB) (regionMap map[string]int, err error) {
	rows, err := db.Queryx("SELECT id, name, city_id FROM regions")
	if err != nil {
		return nil, fmt.Errorf("query regions: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); err == nil && closeErr != nil {
			err = fmt.Errorf("close regions rows: %w", closeErr)
		}
	}()

	regionMap = make(map[string]int)
	for rows.Next() {
		var id, cityID int
		var name string
		if err := rows.Scan(&id, &name, &cityID); err != nil {
			return nil, fmt.Errorf("scan region row: %w", err)
		}
		regionMap[fmt.Sprintf("%d|%s", cityID, name)] = id
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate regions rows: %w", err)
	}
	return regionMap, nil
}

func (c *Cache) GetCityID(city string) int {
	id, exists := c.cityMap[city]
	if !exists {
		return -1
	}
	return id
}

func (c *Cache) GetRegionID(cityID int, region string) int {
	regionKey := fmt.Sprintf("%d|%s", cityID, region)
	id, exists := c.regionMap[regionKey]
	if !exists {
		return -1
	}
	return id
}
