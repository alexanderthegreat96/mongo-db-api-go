package driver

import (
	"testing"
	"time"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestConvertMongoID(t *testing.T) {
	tests := []struct {
		name string
		id   any
	}{
		{
			name: "ObjectID conversion",
			id:   primitive.NewObjectID(),
		},
		{
			name: "String conversion",
			id:   "test_id",
		},
		{
			name: "Int conversion",
			id:   123,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertMongoID(tt.id)
			if result == "" {
				t.Errorf("convertMongoID returned empty string")
			}
		})
	}
}

func TestMongoDBHandler_Where(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		multipleWheres: false,
	}

	handler.Where("name", "=", "John")
	if len(handler.query) == 0 {
		t.Errorf("Where() did not populate query")
	}

	handler2 := &MongoDBHandler{
		query:          make(map[string]any),
		multipleWheres: false,
	}
	handler2.Where("age", ">", 18)
	handler2.Where("status", "=", "active")

	if !handler2.multipleWheres {
		t.Errorf("multipleWheres should be true after 2 Where calls")
	}
}

func TestMongoDBHandler_SortBy(t *testing.T) {
	handler := &MongoDBHandler{
		sort: []primitive.E{},
	}

	handler.SortBy("name", "asc")
	if len(handler.sort) != 1 {
		t.Errorf("expected 1 sort, got %d", len(handler.sort))
	}
	if handler.sort[0].Key != "name" {
		t.Errorf("expected key 'name', got %s", handler.sort[0].Key)
	}
	if handler.sort[0].Value != int32(1) {
		t.Errorf("expected value 1 for asc, got %d", handler.sort[0].Value)
	}

	handler.SortBy("age", "desc")
	if handler.sort[1].Value != int32(-1) {
		t.Errorf("expected value -1 for desc, got %d", handler.sort[1].Value)
	}
}

func TestMongoDBHandler_Pagination(t *testing.T) {
	handler := &MongoDBHandler{
		page:    1,
		perPage: 10,
	}

	handler.Page(2).PerPage(20)
	if handler.page != 2 {
		t.Errorf("expected page 2, got %d", handler.page)
	}
	if handler.perPage != 20 {
		t.Errorf("expected perPage 20, got %d", handler.perPage)
	}

	handler.PerPage(500)
	if handler.perPage != 20 {
		t.Errorf("expected perPage unchanged when > 300, got %d", handler.perPage)
	}

	handler.PerPage(0)
	if handler.perPage != 20 {
		t.Errorf("expected perPage unchanged when <= 0, got %d", handler.perPage)
	}
}

func TestMongoDBHandler_ChunkSlice(t *testing.T) {
	data := make([]map[string]any, 350)
	for i := 0; i < 350; i++ {
		data[i] = map[string]any{"id": i}
	}

	chunks := chunkSlice(data, 300)
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 300 {
		t.Errorf("expected first chunk size 300, got %d", len(chunks[0]))
	}
	if len(chunks[1]) != 50 {
		t.Errorf("expected second chunk size 50, got %d", len(chunks[1]))
	}
}

func TestMongoDBHandler_ChunkInterfaceSlice(t *testing.T) {
	data := make([]any, 350)
	for i := 0; i < 350; i++ {
		data[i] = map[string]any{"id": i}
	}

	chunks := chunkInterfaceSlice(data, 300)
	if len(chunks) != 2 {
		t.Errorf("expected 2 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 300 {
		t.Errorf("expected first chunk size 300, got %d", len(chunks[0]))
	}
	if len(chunks[1]) != 50 {
		t.Errorf("expected second chunk size 50, got %d", len(chunks[1]))
	}
}

func TestMongoDBHandler_ResetState(t *testing.T) {
	handler := &MongoDBHandler{
		query:  map[string]any{"name": "test"},
		sort:   []primitive.E{{Key: "age", Value: 1}},
		page:   2,
		dbName: "testdb",
	}

	handler.ResetState()

	if len(handler.query) != 0 {
		t.Errorf("expected empty query after reset")
	}
	if len(handler.sort) != 0 {
		t.Errorf("expected empty sort after reset")
	}
	if handler.dbName != "testdb" {
		t.Errorf("dbName should persist after ResetState")
	}
}

func TestMongoDBHandler_ResetQuery(t *testing.T) {
	handler := &MongoDBHandler{
		query:          map[string]any{"name": "test"},
		aggregateQuery: []any{bson.M{"$match": bson.M{"status": "active"}}},
		multipleWheres: true,
	}

	handler.ResetQuery()

	if len(handler.query) != 0 {
		t.Errorf("expected empty query after reset")
	}
	if len(handler.aggregateQuery) != 0 {
		t.Errorf("expected empty aggregateQuery after reset")
	}
	if handler.multipleWheres {
		t.Errorf("expected multipleWheres to be false")
	}
}

func TestMongoDBHandler_ResetSort(t *testing.T) {
	handler := &MongoDBHandler{
		sort: []primitive.E{
			{Key: "name", Value: 1},
			{Key: "age", Value: -1},
		},
	}

	handler.ResetSort()

	if len(handler.sort) != 0 {
		t.Errorf("expected empty sort after reset")
	}
}

func TestMongoDBHandler_CountRecords(t *testing.T) {
	tests := []struct {
		name     string
		data     any
		expected int
	}{
		{
			name:     "Slice of maps",
			data:     []map[string]any{make(map[string]any), make(map[string]any), make(map[string]any)},
			expected: 3,
		},
		{
			name:     "Single map",
			data:     map[string]any{"id": 1},
			expected: 1,
		},
		{
			name:     "Slice of interface",
			data:     []any{make(map[string]any), make(map[string]any), make(map[string]any), make(map[string]any)},
			expected: 4,
		},
		{
			name:     "Unsupported type",
			data:     "string",
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := countRecords(tt.data)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestMongoDBHandler_CalculateBatchSize(t *testing.T) {
	tests := []struct {
		name         string
		totalRecords int
		percentage   float64
		expected     int
	}{
		{
			name:         "50% of 100",
			totalRecords: 100,
			percentage:   50,
			expected:     50,
		},
		{
			name:         "25% of 200",
			totalRecords: 200,
			percentage:   25,
			expected:     50,
		},
		{
			name:         "10% of 1000",
			totalRecords: 1000,
			percentage:   10,
			expected:     100,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateBatchSize(tt.totalRecords, tt.percentage)
			if result != tt.expected {
				t.Errorf("expected %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestMongoDBHandler_AppendTimestamps(t *testing.T) {
	handler := &MongoDBHandler{
		useTimestamps: true,
		timeNow:       time.Now(),
	}

	data := map[string]any{"name": "test"}
	result := handler.appendTimestamps(data, "insert")
	resultMap := result.(map[string]any)

	if _, ok := resultMap["created_at"]; !ok {
		t.Errorf("expected created_at field")
	}
	if _, ok := resultMap["updated_at"]; !ok {
		t.Errorf("expected updated_at field")
	}
}

func TestMongoDBHandler_AppendTimestampsDisabled(t *testing.T) {
	handler := &MongoDBHandler{
		useTimestamps: false,
		timeNow:       time.Now(),
	}

	data := map[string]any{"name": "test"}
	result := handler.appendTimestamps(data, "insert")
	resultMap := result.(map[string]any)

	if _, ok := resultMap["created_at"]; ok {
		t.Errorf("should not add timestamps when disabled")
	}
}

func TestMongoDBHandler_DB(t *testing.T) {
	handler := &MongoDBHandler{
		dbName: "olddb",
	}

	result := handler.DB("newdb")

	if result != handler {
		t.Errorf("DB() should return self for chaining")
	}
	if handler.dbName != "newdb" {
		t.Errorf("expected dbName 'newdb', got %s", handler.dbName)
	}
}

func TestMongoDBHandler_Table(t *testing.T) {
	handler := &MongoDBHandler{
		tableName: "oldcol",
	}

	result := handler.Table("newcol")

	if result != handler {
		t.Errorf("Table() should return self for chaining")
	}
	if handler.tableName != "newcol" {
		t.Errorf("expected tableName 'newcol', got %s", handler.tableName)
	}
}

func TestMongoDBHandler_InnerPagePerPage(t *testing.T) {
	handler := &MongoDBHandler{
		innerPage:    1,
		innerPerPage: 10,
	}

	handler.InnerPage(5)
	if handler.innerPage != 5 {
		t.Errorf("expected innerPage 5, got %d", handler.innerPage)
	}

	handler.InnerPerPage(25)
	if handler.innerPerPage != 25 {
		t.Errorf("expected innerPerPage 25, got %d", handler.innerPerPage)
	}

	handler.InnerPerPage(500)
	if handler.innerPerPage != 10 {
		t.Errorf("expected innerPerPage clamped to 10, got %d", handler.innerPerPage)
	}

	handler.InnerPage(0)
	if handler.innerPage != 1 {
		t.Errorf("expected innerPage clamped to 1, got %d", handler.innerPage)
	}
}

func TestMongoDBHandler_OrWhere(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		multipleWheres: false,
	}

	handler.OrWhere("status", "=", "active")
	handler.OrWhere("status", "=", "pending")

	if _, ok := handler.query["$or"]; !ok {
		t.Errorf("expected $or key in query")
	}

	orConditions := handler.query["$or"].([]any)
	if len(orConditions) != 2 {
		t.Errorf("expected 2 OR conditions, got %d", len(orConditions))
	}
}

func TestMongoDBHandler_AndAll(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		multipleWheres: false,
	}

	conditions := [][]any{
		{"age", ">", 18},
		{"status", "=", "active"},
	}

	handler.AndAll(conditions)

	if len(handler.query) == 0 {
		t.Errorf("expected query to be populated")
	}
}

func TestMongoDBHandler_SortAll(t *testing.T) {
	handler := &MongoDBHandler{
		sort: []primitive.E{},
	}

	sorts := [][]any{
		{"name", "asc"},
		{"age", "desc"},
	}

	handler.SortAll(sorts)

	if len(handler.sort) != 2 {
		t.Errorf("expected 2 sorts, got %d", len(handler.sort))
	}
}

func TestMongoDBHandler_GroupBy(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		sort:           []primitive.E{},
		aggregateQuery: []any{},
		page:           1,
		perPage:        10,
		innerPage:      1,
		innerPerPage:   10,
	}

	handler.GroupBy("department")

	if len(handler.aggregateQuery) == 0 {
		t.Errorf("expected aggregateQuery to be populated")
	}

	found := false
	for _, stage := range handler.aggregateQuery {
		if m, ok := stage.(bson.M); ok {
			if _, hasGroup := m["$group"]; hasGroup {
				found = true
				break
			}
		}
	}

	if !found {
		t.Errorf("expected $group stage in pipeline")
	}
}

func TestMongoDBHandler_Limit(t *testing.T) {
	handler := &MongoDBHandler{
		limit: 0,
	}

	result := handler.Limit(100)

	if result != handler {
		t.Errorf("Limit() should return self for chaining")
	}
	if handler.limit != 100 {
		t.Errorf("expected limit 100, got %d", handler.limit)
	}
}

func TestMinInt(t *testing.T) {
	tests := []struct {
		a        int
		b        int
		expected int
	}{
		{5, 10, 5},
		{10, 5, 5},
		{7, 7, 7},
		{-5, 3, -5},
	}

	for _, tt := range tests {
		result := minInt(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("minInt(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestMaxInt(t *testing.T) {
	tests := []struct {
		a        int
		b        int
		expected int
	}{
		{5, 10, 10},
		{10, 5, 10},
		{7, 7, 7},
		{-5, 3, 3},
	}

	for _, tt := range tests {
		result := maxInt(tt.a, tt.b)
		if result != tt.expected {
			t.Errorf("maxInt(%d, %d) = %d, expected %d", tt.a, tt.b, result, tt.expected)
		}
	}
}

func TestMongoDBHandler_NewMongoError(t *testing.T) {
	handler := &MongoDBHandler{
		dbName:    "testdb",
		tableName: "testcol",
		query:     make(map[string]any),
	}

	err := handler.newMongoError(404, "not found")

	if err.Status {
		t.Errorf("expected Status false")
	}
	if err.Code != 404 {
		t.Errorf("expected Code 404, got %d", err.Code)
	}
	if err.Error != "not found" {
		t.Errorf("expected Error 'not found', got %s", err.Error)
	}
	if err.Database != "testdb" {
		t.Errorf("expected Database 'testdb', got %s", err.Database)
	}
}

func TestMongoDBHandler_NewMongoOperations(t *testing.T) {
	handler := &MongoDBHandler{
		dbName:    "testdb",
		tableName: "testcol",
		query:     make(map[string]any),
	}

	result := handler.newMongoOperations(200, true, "insert", "Success")

	if !result.Status {
		t.Errorf("expected Status true")
	}
	if result.Code != 200 {
		t.Errorf("expected Code 200, got %d", result.Code)
	}
	if result.Operation != "insert" {
		t.Errorf("expected Operation 'insert', got %s", result.Operation)
	}
	if result.Message != "Success" {
		t.Errorf("expected Message 'Success', got %s", result.Message)
	}
}

func TestMongoDBHandler_FluentAPI(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		sort:           []primitive.E{},
		page:           1,
		perPage:        10,
		multipleWheres: false,
	}

	result := handler.
		Where("age", ">", 18).
		SortBy("name", "asc").
		Page(2).
		PerPage(20)

	if result != handler {
		t.Errorf("expected fluent API chaining to work")
	}
	if handler.page != 2 {
		t.Errorf("expected page 2, got %d", handler.page)
	}
	if handler.perPage != 20 {
		t.Errorf("expected perPage 20, got %d", handler.perPage)
	}
}

func TestMongoDBHandler_ToJsonBytes(t *testing.T) {
	data := map[string]any{
		"name": "test",
		"age":  30,
	}

	result, err := toJsonBytes(data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == "" {
		t.Errorf("expected non-empty JSON string")
	}
}

func TestMongoDBHandler_Query(t *testing.T) {
	handler := &MongoDBHandler{
		query: map[string]any{"name": "test"},
	}

	result, err := handler.Query()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Errorf("expected non-empty query string")
	}
}

func TestMongoDBHandler_QueryWithAggregation(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		aggregateQuery: []any{bson.M{"$match": bson.M{"status": "active"}}},
	}

	result, err := handler.Query()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if result == "" {
		t.Errorf("expected non-empty query string")
	}
}

func TestMongoDBHandler_QueryNoResults(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		aggregateQuery: []any{},
	}

	result, err := handler.Query()
	if err == nil {
		t.Errorf("expected error for empty query")
	}
	if result != "" {
		t.Errorf("expected empty result string on error")
	}
}

func TestOrderedMap(t *testing.T) {
	om := NewOrderedMap()
	om.AddPair("name", "John")
	om.AddPair("age", 30)

	if om.GetMap() == nil {
		t.Errorf("expected non-nil map")
	}

	json := om.ToJSON()
	if json == "" {
		t.Errorf("expected non-empty JSON")
	}
}

func TestOrderedMapAddPairs(t *testing.T) {
	om := NewOrderedMap()
	pairs := map[string]any{
		"name": "John",
		"age":  30,
	}
	om.AddPairs(pairs)

	json := om.ToJSON()
	if json == "" {
		t.Errorf("expected non-empty JSON after AddPairs")
	}
}

func TestOrderedMapRemovePair(t *testing.T) {
	om := NewOrderedMap()
	om.AddPair("name", "John")
	om.AddPair("age", 30)

	om.RemovePair("age")
	json := om.ToJSON()

	if json == "" {
		t.Errorf("expected non-empty JSON after RemovePair")
	}
}

func TestOrderedMapForEach(t *testing.T) {
	om := NewOrderedMap()
	om.AddPair("a", 1)
	om.AddPair("b", 2)

	count := 0
	om.ForEach(func(key string, value any) {
		count++
	})

	if count != 2 {
		t.Errorf("expected ForEach to iterate 2 times, got %d", count)
	}
}

func TestOrderedMapClear(t *testing.T) {
	om := NewOrderedMap()
	om.AddPair("name", "John")
	om.AddPair("age", 30)

	om.Clear()

	json := om.ToJSON()
	if json != "" {
		t.Errorf("expected empty JSON after Clear, got %s", json)
	}
}

func TestMultipleWhereLogic(t *testing.T) {
	handler := &MongoDBHandler{
		query:          make(map[string]any),
		multipleWheres: false,
	}

	handler.Where("name", "=", "John")
	if handler.multipleWheres {
		t.Errorf("expected multipleWheres false after first Where")
	}

	handler.Where("age", ">", 18)
	if !handler.multipleWheres {
		t.Errorf("expected multipleWheres true after second Where")
	}

	if _, hasAnd := handler.query["$and"]; !hasAnd {
		t.Errorf("expected $and in query after multiple Where calls")
	}
}

func TestSortByDescDefault(t *testing.T) {
	handler := &MongoDBHandler{
		sort: []primitive.E{},
	}

	handler.SortBy("score", "invalid_order")
	if handler.sort[0].Value != int32(-1) {
		t.Errorf("expected default desc (-1) for invalid order, got %d", handler.sort[0].Value)
	}
}
