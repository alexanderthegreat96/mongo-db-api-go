package helpers

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"
)

func GetEnv(key, defaultValue string) string {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	return value
}

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
	operators := []string{"=", "!=", ">=", "<=", "<", ">", "like", "_like_"}

	// slice of slices that cant contain
	// multiple data types
	var parsed [][]interface{}

	if strings.HasPrefix(queryString, "[") && strings.HasSuffix(queryString, "]") {
		queryString = strings.ReplaceAll(queryString, "[", "")
		queryString = strings.ReplaceAll(queryString, "]", "")

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
							operator = strings.ReplaceAll(operator, "_", "")
							parsed = append(parsed, []interface{}{key, operator, convertStringToType(value)})
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
					operator = strings.ReplaceAll(operator, "_", "")
					parsed = append(parsed, []interface{}{key, operator, convertStringToType(value)})
				}
			}
		}

		return parsed
	}

	return nil
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
					if strings.Contains(item, ":") {
						parts := strings.Split(item, ":")
						if len(parts) == 2 {
							parsed = append(parsed, []interface{}{parts[0], parts[1]})
						}
					}
				}
			}
		} else {
			if strings.Contains(queryString, ":") {
				parts := strings.Split(queryString, ":")
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
