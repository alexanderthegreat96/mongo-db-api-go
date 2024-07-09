package helpers

import (
	"encoding/json"

	"github.com/emirpasic/gods/maps/linkedhashmap"
)

func IsValidJSON(s string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(s), &js) == nil
}

type OrderedMap struct {
	Map *linkedhashmap.Map
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Map: linkedhashmap.New(),
	}
}

func (sm *OrderedMap) FromJSON(json string) *OrderedMap {
	if IsValidJSON(json) {
		sm.Map.FromJSON([]byte(json))
	}

	return sm
}

func (sm *OrderedMap) ToJSON() string {
	if !sm.Map.Empty() {
		toJson, _ := sm.Map.ToJSON()
		return string(toJson)
	}
	return ""
}

func (sm *OrderedMap) GetMap() *linkedhashmap.Map {
	return sm.Map
}

func (sm *OrderedMap) AddPair(key string, value interface{}) *OrderedMap {
	sm.Map.Put(key, value)
	return sm
}

func (sm *OrderedMap) RemovePair(key string) *OrderedMap {
	sm.Map.Remove(key)
	return sm
}

func (sm *OrderedMap) AddPairs(data map[string]interface{}) *OrderedMap {
	for key, val := range data {
		sm.Map.Put(key, val)
	}
	return sm
}

func (sm *OrderedMap) ForEach(action func(key string, value interface{})) *OrderedMap {
	sm.Map.Each(func(key, value interface{}) {
		action(key.(string), value)
	})
	return sm
}

func (sm *OrderedMap) Clear() *OrderedMap {
	sm.Map.Clear()

	return sm
}
