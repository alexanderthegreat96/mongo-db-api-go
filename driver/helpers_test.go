package driver

import (
	"testing"

	"go.mongodb.org/mongo-driver/bson"
)

func TestToString(t *testing.T) {
	tests := []struct {
		name     string
		input    any
		expected string
	}{
		{
			name:     "String input",
			input:    "hello",
			expected: "hello",
		},
		{
			name:     "Integer input",
			input:    42,
			expected: "42",
		},
		{
			name:     "Float input",
			input:    3.14,
			expected: "3.14",
		},
		{
			name:     "Boolean input",
			input:    true,
			expected: "true",
		},
		{
			name:     "Nil input",
			input:    nil,
			expected: "<nil>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ToString(tt.input)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestFoundInList(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		list     []string
		expected bool
	}{
		{
			name:     "Found in list",
			input:    "apple",
			list:     []string{"apple", "banana", "orange"},
			expected: true,
		},
		{
			name:     "Not found in list",
			input:    "grape",
			list:     []string{"apple", "banana", "orange"},
			expected: false,
		},
		{
			name:     "Empty list",
			input:    "apple",
			list:     []string{},
			expected: false,
		},
		{
			name:     "Single element found",
			input:    "test",
			list:     []string{"test"},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := foundInList(tt.input, tt.list)
			if result != tt.expected {
				t.Errorf("expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestConvertStringToType(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		checkFn  func(any) bool
		typeDesc string
	}{
		{
			name:     "null converts to nil",
			input:    "null",
			checkFn:  func(v any) bool { return v == nil },
			typeDesc: "nil",
		},
		{
			name:     "true converts to bool",
			input:    "true",
			checkFn:  func(v any) bool { _, ok := v.(bool); return ok && v == true },
			typeDesc: "bool",
		},
		{
			name:     "false converts to bool",
			input:    "false",
			checkFn:  func(v any) bool { _, ok := v.(bool); return ok && v == false },
			typeDesc: "bool",
		},
		{
			name:     "Integer converts to int",
			input:    "42",
			checkFn:  func(v any) bool { val, ok := v.(int); return ok && val == 42 },
			typeDesc: "int",
		},
		{
			name:     "Large integer converts to int64",
			input:    "9999999999",
			checkFn:  func(v any) bool { _, ok := v.(int64); return ok },
			typeDesc: "int64",
		},
		{
			name:     "Float converts to float64",
			input:    "3.14",
			checkFn:  func(v any) bool { val, ok := v.(float64); return ok && val > 3.1 && val < 3.2 },
			typeDesc: "float64",
		},
		{
			name:     "Plain string returns string",
			input:    "hello",
			checkFn:  func(v any) bool { val, ok := v.(string); return ok && val == "hello" },
			typeDesc: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertStringToType(tt.input)
			if !tt.checkFn(result) {
				t.Errorf("expected %s, got %v", tt.typeDesc, result)
			}
		})
	}
}

func TestProcessValue(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		autoConv bool
		checkFn  func(any) bool
		typeDesc string
	}{
		{
			name:     "Quoted string returns unquoted",
			input:    `"hello"`,
			autoConv: true,
			checkFn:  func(v any) bool { return v == "hello" },
			typeDesc: "unquoted string",
		},
		{
			name:     "Single quoted string returns unquoted",
			input:    `'world'`,
			autoConv: true,
			checkFn:  func(v any) bool { return v == "world" },
			typeDesc: "unquoted string",
		},
		{
			name:     "String with /a flag auto converts",
			input:    "42/a",
			autoConv: false,
			checkFn:  func(v any) bool { val, ok := v.(int); return ok && val == 42 },
			typeDesc: "auto-converted int",
		},
		{
			name:     "String with /n flag disables auto convert",
			input:    "42/n",
			autoConv: true,
			checkFn:  func(v any) bool { val, ok := v.(string); return ok && val == "42" },
			typeDesc: "non-converted string",
		},
		{
			name:     "Auto convert enabled converts number",
			input:    "123",
			autoConv: true,
			checkFn:  func(v any) bool { val, ok := v.(int); return ok && val == 123 },
			typeDesc: "int",
		},
		{
			name:     "Auto convert disabled returns string",
			input:    "456",
			autoConv: false,
			checkFn:  func(v any) bool { return v == "456" },
			typeDesc: "string",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processValue(tt.input, tt.autoConv)
			if !tt.checkFn(result) {
				t.Errorf("expected %s, got %v", tt.typeDesc, result)
			}
		})
	}
}

func TestParseQuery(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		autoConv bool
		checkLen int
		checkKey string
		checkOp  string
	}{
		{
			name:     "Simple equality query",
			input:    "name,=,John",
			autoConv: true,
			checkLen: 1,
			checkKey: "name",
			checkOp:  "=",
		},
		{
			name:     "Multiple OR conditions",
			input:    "status,=,active|status,=,pending",
			autoConv: true,
			checkLen: 2,
			checkKey: "status",
			checkOp:  "=",
		},
		{
			name:     "Between operator",
			input:    "age,between,[18:65]",
			autoConv: true,
			checkLen: 1,
			checkKey: "age",
			checkOp:  "between",
		},
		{
			name:     "Like operator with underscore normalization",
			input:    "email,_like_,test",
			autoConv: true,
			checkLen: 1,
			checkKey: "email",
			checkOp:  "like",
		},
		{
			name:     "Query with brackets",
			input:    "[age,>,18]",
			autoConv: true,
			checkLen: 1,
			checkKey: "age",
			checkOp:  ">",
		},
		{
			name:     "Invalid operator ignored",
			input:    "name,invalid_op,test",
			autoConv: true,
			checkLen: 0,
		},
		{
			name:     "Wrong parts count ignored",
			input:    "name,test",
			autoConv: true,
			checkLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseQuery(tt.input, tt.autoConv)
			if len(result) != tt.checkLen {
				t.Errorf("expected %d parsed queries, got %d", tt.checkLen, len(result))
			}

			if tt.checkLen > 0 && len(result) > 0 {
				key := result[0][0].(string)
				op := result[0][1].(string)

				if key != tt.checkKey {
					t.Errorf("expected key %s, got %s", tt.checkKey, key)
				}
				if op != tt.checkOp {
					t.Errorf("expected operator %s, got %s", tt.checkOp, op)
				}
			}
		})
	}
}

func TestParseSort(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{
			name:     "Single sort with colon",
			input:    "[name:asc]",
			expected: 1,
		},
		{
			name:     "Multiple sorts with pipe",
			input:    "[name:asc|age:desc]",
			expected: 2,
		},
		{
			name:     "Sort with comma",
			input:    "[email,desc]",
			expected: 1,
		},
		{
			name:     "Multiple sorts mixed",
			input:    "[name:asc|age,desc]",
			expected: 2,
		},
		{
			name:     "No brackets returns nil",
			input:    "name:asc",
			expected: 0,
		},
		{
			name:     "Invalid format returns nil",
			input:    "[invalid]",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ParseSort(tt.input)
			if result == nil && tt.expected == 0 {
				return
			}
			if result == nil {
				t.Errorf("expected %d sorts, got nil", tt.expected)
				return
			}
			if len(result) != tt.expected {
				t.Errorf("expected %d sorts, got %d", tt.expected, len(result))
			}
		})
	}
}

func TestConvertJsonToData(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
	}{
		{
			name:      "Valid JSON object",
			input:     `{"name":"test"}`,
			shouldErr: false,
		},
		{
			name:      "Valid JSON array",
			input:     `[1,2,3]`,
			shouldErr: false,
		},
		{
			name:      "Valid JSON string",
			input:     `"hello"`,
			shouldErr: false,
		},
		{
			name:      "Invalid JSON",
			input:     `{invalid}`,
			shouldErr: true,
		},
		{
			name:      "Empty string",
			input:     ``,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertJsonToData(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got error=%v", tt.shouldErr, err != nil)
			}
			if !tt.shouldErr && result == nil {
				t.Errorf("expected non-nil result for valid JSON")
			}
		})
	}
}

func TestConvertJsonToMap(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		shouldErr bool
		expectLen int
	}{
		{
			name:      "Valid JSON object",
			input:     `{"name":"test","age":30}`,
			shouldErr: false,
			expectLen: 2,
		},
		{
			name:      "Empty JSON object",
			input:     `{}`,
			shouldErr: false,
			expectLen: 0,
		},
		{
			name:      "Invalid JSON",
			input:     `{invalid}`,
			shouldErr: true,
		},
		{
			name:      "JSON array not object",
			input:     `[1,2,3]`,
			shouldErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ConvertJsonToMap(tt.input)
			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got error=%v", tt.shouldErr, err != nil)
			}
			if !tt.shouldErr && len(result) != tt.expectLen {
				t.Errorf("expected %d fields, got %d", tt.expectLen, len(result))
			}
		})
	}
}

func TestConvertMapToJsonOrdered(t *testing.T) {
	m := map[string]any{
		"name": "John",
		"age":  30,
	}

	result, err := ConvertMapToJsonOrdered(m)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == "" {
		t.Errorf("expected non-empty JSON string")
	}

	if !contains(result, "John") || !contains(result, "30") {
		t.Errorf("expected JSON to contain data")
	}
}

func TestAppendCreatedAtToJson(t *testing.T) {
	jsonStr := `{"name":"test"}`
	result := AppendCreatedAtToJson(jsonStr)

	if len(result) == 0 {
		t.Errorf("expected non-empty bson.D")
	}

	hasCreatedAt := false
	hasUpdatedAt := false

	for _, elem := range result {
		if elem.Key == "created_at" {
			hasCreatedAt = true
		}
		if elem.Key == "updated_at" {
			hasUpdatedAt = true
		}
	}

	if !hasCreatedAt {
		t.Errorf("expected created_at field")
	}
	if !hasUpdatedAt {
		t.Errorf("expected updated_at field")
	}
}

func TestAppendUpdatedAtToJson(t *testing.T) {
	jsonStr := `{"name":"test"}`
	result := AppendUpdatedAtToJson(jsonStr)

	if len(result) == 0 {
		t.Errorf("expected non-empty bson.D")
	}

	hasUpdatedAt := false

	for _, elem := range result {
		if elem.Key == "updated_at" {
			hasUpdatedAt = true
		}
	}

	if !hasUpdatedAt {
		t.Errorf("expected updated_at field")
	}
}

func TestMapOperators(t *testing.T) {
	tests := []struct {
		name       string
		operator   string
		value      any
		shouldErr  bool
		hasBsonKey string
	}{
		{
			name:       "Equality operator",
			operator:   "=",
			value:      "test",
			shouldErr:  false,
			hasBsonKey: "$eq",
		},
		{
			name:       "Not equal operator",
			operator:   "!=",
			value:      "test",
			shouldErr:  false,
			hasBsonKey: "$ne",
		},
		{
			name:       "Greater than operator",
			operator:   ">",
			value:      18,
			shouldErr:  false,
			hasBsonKey: "$gt",
		},
		{
			name:       "Like operator",
			operator:   "like",
			value:      "test",
			shouldErr:  false,
			hasBsonKey: "$regex",
		},
		{
			name:       "Between operator",
			operator:   "between",
			value:      []any{10, 20},
			shouldErr:  false,
			hasBsonKey: "$gte",
		},
		{
			name:       "Unknown operator",
			operator:   "invalid_op",
			value:      "test",
			shouldErr:  true,
			hasBsonKey: "",
		},
		{
			name:       "Between with invalid value",
			operator:   "between",
			value:      "not_a_slice",
			shouldErr:  true,
			hasBsonKey: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := MapOperators(tt.operator, tt.value)

			if (err != nil) != tt.shouldErr {
				t.Errorf("expected error=%v, got error=%v", tt.shouldErr, err != nil)
			}

			if !tt.shouldErr {
				if resultMap, ok := result.(bson.M); ok {
					if _, hasKey := resultMap[tt.hasBsonKey]; !hasKey {
						t.Errorf("expected key %s in result", tt.hasBsonKey)
					}
				} else {
					t.Errorf("expected bson.M result")
				}
			}
		})
	}
}

func TestMapOperatorsEquivalent(t *testing.T) {
	result1, err1 := MapOperators("!=", "test")
	result2, err2 := MapOperators("<>", "test")

	if err1 != nil || err2 != nil {
		t.Errorf("expected no errors")
	}

	if resultMap1, ok := result1.(bson.M); ok {
		if _, hasKey := resultMap1["$ne"]; !hasKey {
			t.Errorf("!= operator should map to $ne")
		}
	}

	if resultMap2, ok := result2.(bson.M); ok {
		if _, hasKey := resultMap2["$ne"]; !hasKey {
			t.Errorf("<> operator should map to $ne")
		}
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
