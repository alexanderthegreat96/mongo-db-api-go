package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/alexanderthegreat96/go-ordered-map/omap"
	"go.mongodb.org/mongo-driver/bson"
)

func ToString(v interface{}) string {
	return fmt.Sprint(v)
}

func foundInList(input string, list []string) bool {
	if len(list) > 0 {
		for _, v := range list {
			if v == input {
				return true
			}
		}
	}
	return false
}

func convertStringToType(value string) interface{} {
	if intValue, err := strconv.Atoi(value); err == nil {
		return intValue
	}
	if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
		return floatValue
	}
	return value
}

func ParseQuery(queryString string) [][]interface{} {
	operators := []string{
		"=",
		"!=",
		"<>",
		"<",
		"<=",
		">",
		">=",
		"like",
		"_like_",
		"_i_like_",
		"not_like",
		"ilike",
		"&",
		"|",
		"^",
		"<<",
		">>",
		"rlike",
		"regexp",
		"not_regexp",
		"exists",
		"type",
		"mod",
		"where",
		"all",
		"size",
		"regex",
		"not_regex",
		"text",
		"slice",
		"elemmatch",
		"geowithin",
		"geointersects",
		"near",
		"nearsphere",
		"geometry",
		"maxdistance",
		"center",
		"centersphere",
		"box",
		"polygon",
		"uniquedocs",
		"between",
	}

	var parsed [][]interface{}

	// only removing the outer brackets
	if strings.HasPrefix(queryString, "[") && strings.HasSuffix(queryString, "]") {
		queryString = queryString[1 : len(queryString)-1]
	}

	if strings.Contains(queryString, "|") {
		items := strings.Split(queryString, "|")
		if len(items) > 0 {
			for _, item := range items {
				parts := strings.Split(item, ",")
				if len(parts) == 3 {
					key := strings.TrimSpace(ToString(parts[0]))
					operator := strings.TrimSpace(ToString(parts[1]))
					value := strings.TrimSpace(ToString(parts[2]))

					if foundInList(operator, operators) {
						if operator == "between" {
							value = strings.ReplaceAll(value, "[", "")
							value = strings.ReplaceAll(value, "]", "")
							betweenParts := strings.Split(value, ":")
							if len(betweenParts) == 2 {
								low := convertStringToType(strings.TrimSpace(betweenParts[0]))
								high := convertStringToType(strings.TrimSpace(betweenParts[1]))
								parsed = append(parsed, []interface{}{key, operator, []interface{}{low, high}})
							}
						} else {
							operator = strings.ReplaceAll(operator, "_", "")
							parsed = append(parsed, []interface{}{key, operator, convertStringToType(value)})
						}
					}
				}
			}
		}
	} else {
		parts := strings.Split(queryString, ",")
		if len(parts) == 3 {
			key := strings.TrimSpace(ToString(parts[0]))
			operator := strings.TrimSpace(ToString(parts[1]))
			value := strings.TrimSpace(ToString(parts[2]))

			if foundInList(operator, operators) {
				if operator == "between" {
					value = strings.ReplaceAll(value, "[", "")
					value = strings.ReplaceAll(value, "]", "")
					betweenParts := strings.Split(value, ":")
					if len(betweenParts) == 2 {
						low := convertStringToType(strings.TrimSpace(betweenParts[0]))
						high := convertStringToType(strings.TrimSpace(betweenParts[1]))
						parsed = append(parsed, []interface{}{key, operator, []interface{}{low, high}})
					}
				} else {
					operator = strings.ReplaceAll(operator, "_", "")
					parsed = append(parsed, []interface{}{key, operator, convertStringToType(value)})
				}
			}
		}
	}

	return parsed
}

func ParseSort(queryString string) [][]interface{} {
	var parsed [][]interface{}
	if strings.HasPrefix(queryString, "[") && strings.HasSuffix(queryString, "]") {
		queryString = strings.ReplaceAll(queryString, "[", "")
		queryString = strings.ReplaceAll(queryString, "]", "")

		if strings.Contains(queryString, "|") {
			items := strings.Split(queryString, "|")
			if len(items) > 0 {
				for _, item := range items {
					// support for [field_name: desc]
					if strings.Contains(item, ":") {
						parts := strings.Split(item, ":")
						if len(parts) == 2 {
							parsed = append(parsed, []interface{}{parts[0], parts[1]})
						}
					}
					// support for [field_name, desc]
					if strings.Contains(item, ",") {
						parts := strings.Split(item, ",")
						if len(parts) == 2 {
							parsed = append(parsed, []interface{}{parts[0], parts[1]})
						}
					}
				}
			}
		} else {
			// support for [field_name: desc]
			if strings.Contains(queryString, ":") {
				parts := strings.Split(queryString, ":")
				if len(parts) == 2 {
					parsed = append(parsed, []interface{}{parts[0], parts[1]})
				}
			}

			// support for [field_name, desc]
			if strings.Contains(queryString, ",") {
				parts := strings.Split(queryString, ",")
				if len(parts) == 2 {
					parsed = append(parsed, []interface{}{parts[0], parts[1]})
				}
			}
		}
		return parsed
	}

	return nil
}

func ConvertJsonToData(jsonInput string) (interface{}, error) {
	var result interface{}
	err := json.Unmarshal([]byte(jsonInput), &result)
	if err != nil {
		return nil, errors.New("the input string is not a valid JSON")
	}
	return result, nil
}

func ConvertJsonToMap(jsonInput string) (map[string]interface{}, error) {
	var result map[string]interface{}
	err := json.Unmarshal([]byte(jsonInput), &result)
	if err != nil {
		return nil, errors.New("the input string is not a valid JSON")
	}
	return result, nil
}

func ConvertMapToJsonOrdered(m map[string]interface{}) (string, error) {

	om := omap.NewOrderedMap()
	for key, value := range m {
		om.AddPair(key, value)
	}

	jsonBytes, err := om.Map.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("error converting ordered map to JSON: %v", err)
	}

	// Convert JSON bytes to string and return
	return string(jsonBytes), nil
}

func AppendCreatedAtToJson(jsonStr string) bson.D {
	// new ordered map
	om := omap.NewOrderedMap()
	// populate existing data from json
	om.FromJSON(jsonStr)
	// adding timestamps
	om.AddPair("created_at", time.Now())
	om.AddPair("updated_at", time.Now())

	// converting back to json
	json := om.ToJSON()

	// converting to an ordered bson
	var result bson.D
	if err := bson.UnmarshalExtJSON([]byte(json), true, &result); err != nil {
		return nil
	}

	return result
}

func AppendUpdatedAtToJson(jsonStr string) bson.D {
	// new ordered map
	om := omap.NewOrderedMap()
	// populate existing data from json
	om.FromJSON(jsonStr)
	// adding timestamps
	om.AddPair("updated_at", time.Now())

	// converting back to json
	json := om.ToJSON()

	// converting to an ordered bson
	var result bson.D
	if err := bson.UnmarshalExtJSON([]byte(json), true, &result); err != nil {
		return nil
	}

	return result
}

func MapOperators(operator string, value any) (any, error) {
	operatorMap := map[string]any{
		"=":             bson.M{"$eq": value},
		"!=":            bson.M{"$ne": value},
		"<>":            bson.M{"$ne": value},
		"<":             bson.M{"$lt": value},
		"<=":            bson.M{"$lte": value},
		">":             bson.M{"$gt": value},
		">=":            bson.M{"$gte": value},
		"like":          bson.M{"$regex": value, "$options": "i"},
		"not_like":      bson.M{"$not": bson.M{"$regex": value}},
		"ilike":         bson.M{"$regex": value, "$options": "i"},
		"&":             bson.M{"$bitsAllSet": value},
		"|":             bson.M{"$bitsAnySet": value},
		"^":             bson.M{"$bitsAllClear": value},
		"<<":            bson.M{"$bitsAllClear": value},
		">>":            bson.M{"$bitsAnyClear": value},
		"rlike":         bson.M{"$regex": value},
		"regexp":        bson.M{"$regex": value},
		"not_regexp":    bson.M{"$not": bson.M{"$regex": value}, "$options": "i"},
		"exists":        bson.M{"$exists": value},
		"type":          bson.M{"$type": value},
		"mod":           bson.M{"$mod": value},
		"where":         bson.M{"$where": value},
		"all":           bson.M{"$all": value},
		"size":          bson.M{"$size": value},
		"regex":         bson.M{"$regex": value},
		"not_regex":     bson.M{"$not": bson.M{"$regex": value}, "$options": "i"},
		"text":          bson.M{"$text": value},
		"slice":         bson.M{"$slice": value},
		"elemmatch":     bson.M{"$elemMatch": value},
		"geowithin":     bson.M{"$geoWithin": value},
		"geointersects": bson.M{"$geoIntersects": value},
		"near":          bson.M{"$near": value},
		"nearsphere":    bson.M{"$nearSphere": value},
		"geometry":      bson.M{"$geometry": value},
		"maxdistance":   bson.M{"$maxDistance": value},
		"center":        bson.M{"$center": value},
		"centersphere":  bson.M{"$centerSphere": value},
		"box":           bson.M{"$box": value},
		"polygon":       bson.M{"$polygon": value},
		"uniquedocs":    bson.M{"$uniqueDocs": value},
	}
	// Special handling for "between".
	if operator == "between" {
		v, ok := value.([]any)
		if !ok || len(v) != 2 {
			return nil, errors.New("value must be a slice with exactly two elements for 'between'")
		}
		return bson.M{"$gte": v[0], "$lte": v[1]}, nil
	}
	mappedValue, exists := operatorMap[operator]
	if !exists {
		return nil, fmt.Errorf("unknown operator: %s", operator)
	}
	return mappedValue, nil
}
