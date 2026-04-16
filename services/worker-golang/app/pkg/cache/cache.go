package cache

import (
	"fmt"

	"github.com/jmoiron/sqlx"
)

type Cache struct {
	cityMap   map[string]int
	regionMap map[string]int
}

func DebugString(label, s string) {
	fmt.Printf("%s : [%s] HEX: [% x]\n", label, s, []byte(s))
}

func NewCache(db *sqlx.DB) *Cache {
	cityRows, _ := db.Queryx("SELECT id, name FROM cities")
	cityMap := make(map[string]int)
	for cityRows.Next() {
		var id int
		var name string
		cityRows.Scan(&id, &name)
		cityMap[name] = id
		// DebugString("City", name)
	}
	// caching region primary keys
	regionRows, _ := db.Queryx("SELECT id, name, city_id FROM regions")
	regionMap := make(map[string]int)
	for regionRows.Next() {
		var id, cityID int
		var name string
		regionRows.Scan(&id, &name, &cityID)
		regionMap[fmt.Sprintf("%d|%s", cityID, name)] = id
	}
	return &Cache{
		cityMap:   cityMap,
		regionMap: regionMap,
	}
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
