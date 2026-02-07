package driver

import (
	"encoding/json"
	"errors"
	"fmt"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/alexanderthegreat96/go-ordered-map/omap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func ToString(v any) string {
	return fmt.Sprint(v)
}

func foundInList(input string, list []string) bool {
	if len(list) > 0 {
		if slices.Contains(list, input) {
			return true
		}
	}
	return false
}

func convertStringToType(value string) any {
	val := strings.ToLower(strings.TrimSpace(value))

	if val == "null" {
		return nil
	}

	if val == "true" {
		return true
	}
	if val == "false" {
		return false
	}

	if int64Val, err := strconv.ParseInt(value, 10, 64); err == nil {
		if int64Val <= int64(^uint32(0)>>1) {
			return int(int64Val)
		}
		return int64Val
	}

	if floatVal, err := strconv.ParseFloat(value, 64); err == nil {
		return floatVal
	}

	if t, err := time.Parse(time.RFC3339, value); err == nil {
		return t
	}

	if ts, err := strconv.ParseInt(value, 10, 64); err == nil {
		if ts > 1e12 {
			return time.UnixMilli(ts)
		}
		return time.Unix(ts, 0)
	}

	if oid, err := primitive.ObjectIDFromHex(value); err == nil {
		return oid
	}

	var jsonResult any
	if err := json.Unmarshal([]byte(value), &jsonResult); err == nil {
		return jsonResult
	}

	return value
}

// will process the value
// applying auto-conversion
// if specifically stated or not
func processValue(raw string, autoConversion bool) any {
	raw = strings.TrimSpace(raw)

	useAutoConvert := autoConversion
	if strings.HasSuffix(raw, `/a`) {
		useAutoConvert = true
		raw = strings.TrimSpace(raw[:len(raw)-2])
	} else if strings.HasSuffix(raw, `/n`) {
		useAutoConvert = false
		raw = strings.TrimSpace(raw[:len(raw)-2])
	}

	if len(raw) >= 2 && ((raw[0] == '"' && raw[len(raw)-1] == '"') ||
		(raw[0] == '\'' && raw[len(raw)-1] == '\'')) {
		return raw[1 : len(raw)-1]
	}

	if useAutoConvert {
		return convertStringToType(raw)
	}
	return raw

}
func ParseQuery(queryString string, autoConvertTypes bool) [][]any {
	operators := []string{
		"=", "!=", "<>", "<", "<=", ">", ">=",
		"like", "_like_", "_i_like_", "not_like", "ilike",
		"&", "|", "^", "<<", ">>",
		"rlike", "regexp", "not_regexp",
		"exists", "type", "mod", "where",
		"all", "size", "regex", "not_regex", "text",
		"slice", "elemmatch",
		"geowithin", "geointersects",
		"near", "nearsphere", "geometry", "maxdistance",
		"center", "centersphere", "box", "polygon",
		"uniquedocs", "between",
	}

	var parsed [][]any

	if strings.HasPrefix(queryString, "[") && strings.HasSuffix(queryString, "]") {
		queryString = queryString[1 : len(queryString)-1]
	}

	clauses := []string{queryString}
	if strings.Contains(queryString, "|") {
		clauses = strings.Split(queryString, "|")
	}

	for _, clause := range clauses {
		parts := strings.Split(clause, ",")
		if len(parts) != 3 {
			continue
		}

		key := strings.TrimSpace(ToString(parts[0]))
		op := strings.TrimSpace(ToString(parts[1]))
		raw := strings.TrimSpace(ToString(parts[2]))

		if !foundInList(op, operators) {
			continue
		}

		if op == "between" {
			cleaned := strings.Trim(raw, "[]")
			endpoints := strings.SplitN(cleaned, ":", 2)
			if len(endpoints) == 2 {
				low := processValue(endpoints[0], autoConvertTypes)
				high := processValue(endpoints[1], autoConvertTypes)
				parsed = append(parsed, []any{key, op, []any{low, high}})
			}
		} else {
			op = strings.ReplaceAll(op, "_", "")
			val := processValue(raw, autoConvertTypes)
			parsed = append(parsed, []any{key, op, val})
		}
	}

	return parsed
}

func ParseSort(queryString string) [][]any {
	var parsed [][]any
	if strings.HasPrefix(queryString, "[") && strings.HasSuffix(queryString, "]") {
		queryString = strings.ReplaceAll(queryString, "[", "")
		queryString = strings.ReplaceAll(queryString, "]", "")

		if strings.Contains(queryString, "|") {
			items := strings.Split(queryString, "|")
			if len(items) > 0 {
				for _, item := range items {
					if strings.Contains(item, ":") {
						parts := strings.Split(item, ":")
						if len(parts) == 2 {
							parsed = append(parsed, []any{parts[0], parts[1]})
						}
					}
					if strings.Contains(item, ",") {
						parts := strings.Split(item, ",")
						if len(parts) == 2 {
							parsed = append(parsed, []any{parts[0], parts[1]})
						}
					}
				}
			}
		} else {
			if strings.Contains(queryString, ":") {
				parts := strings.Split(queryString, ":")
				if len(parts) == 2 {
					parsed = append(parsed, []any{parts[0], parts[1]})
				}
			}

			if strings.Contains(queryString, ",") {
				parts := strings.Split(queryString, ",")
				if len(parts) == 2 {
					parsed = append(parsed, []any{parts[0], parts[1]})
				}
			}
		}
		return parsed
	}

	return nil
}

func ConvertJsonToData(jsonInput string) (any, error) {
	var result any
	err := json.Unmarshal([]byte(jsonInput), &result)
	if err != nil {
		return nil, errors.New("the input string is not a valid JSON")
	}
	return result, nil
}

func ConvertJsonToMap(jsonInput string) (map[string]any, error) {
	var result map[string]any
	err := json.Unmarshal([]byte(jsonInput), &result)
	if err != nil {
		return nil, errors.New("the input string is not a valid JSON")
	}
	return result, nil
}

func ConvertMapToJsonOrdered(m map[string]any) (string, error) {

	om := omap.NewOrderedMap()
	for key, value := range m {
		om.AddPair(key, value)
	}

	jsonBytes, err := om.Map.MarshalJSON()
	if err != nil {
		return "", fmt.Errorf("error converting ordered map to JSON: %v", err)
	}

	return string(jsonBytes), nil
}

func AppendCreatedAtToJson(jsonStr string) bson.D {
	om := omap.NewOrderedMap().FromJSON(jsonStr)

	now := time.Now()
	om.AddPair("created_at", now)
	om.AddPair("updated_at", now)

	var result bson.D
	om.ForEach(func(key string, value any) {
		result = append(result, bson.E{Key: key, Value: value})
	})
	return result
}

func AppendUpdatedAtToJson(jsonStr string) bson.D {
	om := omap.NewOrderedMap().FromJSON(jsonStr)

	now := time.Now()
	om.AddPair("updated_at", now)

	var result bson.D
	om.ForEach(func(key string, value any) {
		result = append(result, bson.E{Key: key, Value: value})
	})
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

// older implementation
// i left it here in case I need to
// go back to it, although, probably not
//func ParseQuery(queryString string, autoConvertTypes bool) [][]any {
//	operators := []string{
//		"=",
//		"!=",
//		"<>",
//		"<",
//		"<=",
//		">",
//		">=",
//		"like",
//		"_like_",
//		"_i_like_",
//		"not_like",
//		"ilike",
//		"&",
//		"|",
//		"^",
//		"<<",
//		">>",
//		"rlike",
//		"regexp",
//		"not_regexp",
//		"exists",
//		"type",
//		"mod",
//		"where",
//		"all",
//		"size",
//		"regex",
//		"not_regex",
//		"text",
//		"slice",
//		"elemmatch",
//		"geowithin",
//		"geointersects",
//		"near",
//		"nearsphere",
//		"geometry",
//		"maxdistance",
//		"center",
//		"centersphere",
//		"box",
//		"polygon",
//		"uniquedocs",
//		"between",
//	}
//
//	var parsed [][]any
//
//	// only removing the outer brackets
//	if strings.HasPrefix(queryString, "[") && strings.HasSuffix(queryString, "]") {
//		queryString = queryString[1 : len(queryString)-1]
//	}
//
//	if strings.Contains(queryString, "|") {
//		items := strings.Split(queryString, "|")
//		if len(items) > 0 {
//			for _, item := range items {
//				parts := strings.Split(item, ",")
//				if len(parts) == 3 {
//					key := strings.TrimSpace(ToString(parts[0]))
//					operator := strings.TrimSpace(ToString(parts[1]))
//					value := strings.TrimSpace(ToString(parts[2]))
//
//					if foundInList(operator, operators) {
//						if operator == "between" {
//							value = strings.ReplaceAll(value, "[", "")
//							value = strings.ReplaceAll(value, "]", "")
//							betweenParts := strings.Split(value, ":")
//							if len(betweenParts) == 2 {
//								low := convertStringToType(strings.TrimSpace(betweenParts[0]))
//								high := convertStringToType(strings.TrimSpace(betweenParts[1]))
//								parsed = append(parsed, []any{key, operator, []any{low, high}})
//							}
//						} else {
//							operator = strings.ReplaceAll(operator, "_", "")
//							parsed = append(parsed, []any{key, operator, convertStringToType(value)})
//						}
//					}
//				}
//			}
//		}
//	} else {
//		parts := strings.Split(queryString, ",")
//		if len(parts) == 3 {
//			key := strings.TrimSpace(ToString(parts[0]))
//			operator := strings.TrimSpace(ToString(parts[1]))
//			value := strings.TrimSpace(ToString(parts[2]))
//
//			if foundInList(operator, operators) {
//				if operator == "between" {
//					value = strings.ReplaceAll(value, "[", "")
//					value = strings.ReplaceAll(value, "]", "")
//					betweenParts := strings.Split(value, ":")
//					if len(betweenParts) == 2 {
//						low := convertStringToType(strings.TrimSpace(betweenParts[0]))
//						high := convertStringToType(strings.TrimSpace(betweenParts[1]))
//						parsed = append(parsed, []any{key, operator, []any{low, high}})
//					}
//				} else {
//					operator = strings.ReplaceAll(operator, "_", "")
//					parsed = append(parsed, []any{key, operator, convertStringToType(value)})
//				}
//			}
//		}
//	}
//
//	return parsed
//}
