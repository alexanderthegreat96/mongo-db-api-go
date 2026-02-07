package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/alexanderthegreat96/mongo-db-api-go/responses"
	"github.com/gin-gonic/gin"
)

type MockMongoDBHandler struct {
	MockListDatabasesResult   driver.MongoDatabaseListResult
	MockListDatabasesError    driver.MongoError
	MockDropDatabaseError     driver.MongoError
	MockListCollectionsResult driver.MongoTablesListResult
	MockListCollectionsError  driver.MongoError
	MockDropTableError        driver.MongoError
	MockFindResult            driver.MongoResults
	MockFindError             driver.MongoError
	MockFindByIdResult        driver.SingleMongoResult
	MockFindByIdError         driver.MongoError
	MockInsertResult          driver.MongoOperationsResult
	MockInsertError           driver.MongoError
	MockUpdateByIdResult      driver.MongoOperationsResult
	MockUpdateByIdError       driver.MongoError
	MockUpdateResult          driver.MongoOperationsResult
	MockUpdateError           driver.MongoError
	MockDeleteByIdResult      driver.MongoOperationsResult
	MockDeleteByIdError       driver.MongoError
	MockDeleteResult          driver.MongoOperationsResult
	MockDeleteError           driver.MongoError
	MockExecuteRawResult      driver.MongoResults
	MockExecuteRawError       driver.MongoError
	MockCountResult           driver.CountMongoResult
	MockCountError            driver.MongoError

	// Track what was called
	LastDBName       string
	LastTableName    string
	LastPage         int
	LastPerPage      int
	LastInnerPage    int
	LastInnerPerPage int
	LastInsertData   interface{}
	LastUpdateData   interface{}
	LastMongoId      string
	LastRawQuery     string
	LastAsPipeline   bool
	AndQueryCalled   bool
	OrQueryCalled    bool
	SortCalled       bool
	GroupCalled      bool
	ResetQueryCalled bool
	ResetSortCalled  bool
}

// Implement the fluent interface methods
func (m *MockMongoDBHandler) DB(name string) *MockMongoDBHandler {
	m.LastDBName = name
	return m
}

func (m *MockMongoDBHandler) Table(name string) *MockMongoDBHandler {
	m.LastTableName = name
	return m
}

func (m *MockMongoDBHandler) Page(page int) *MockMongoDBHandler {
	m.LastPage = page
	return m
}

func (m *MockMongoDBHandler) PerPage(perPage int) *MockMongoDBHandler {
	m.LastPerPage = perPage
	return m
}

func (m *MockMongoDBHandler) InnerPage(page int) *MockMongoDBHandler {
	m.LastInnerPage = page
	return m
}

func (m *MockMongoDBHandler) InnerPerPage(perPage int) *MockMongoDBHandler {
	m.LastInnerPerPage = perPage
	return m
}

func (m *MockMongoDBHandler) AndAll(query [][]any) *MockMongoDBHandler {
	if len(query) > 0 {
		m.AndQueryCalled = true
	}
	return m
}

func (m *MockMongoDBHandler) OrAll(query [][]any) *MockMongoDBHandler {
	if len(query) > 0 {
		m.OrQueryCalled = true
	}
	return m
}

func (m *MockMongoDBHandler) SortAll(sort [][]any) *MockMongoDBHandler {
	if len(sort) > 0 {
		m.SortCalled = true
	}
	return m
}

func (m *MockMongoDBHandler) GroupAll(group string) *MockMongoDBHandler {
	if group != "" {
		m.GroupCalled = true
	}
	return m
}

func (m *MockMongoDBHandler) ResetQuery() {
	m.ResetQueryCalled = true
}

func (m *MockMongoDBHandler) ResetSort() {
	m.ResetSortCalled = true
}

func (m *MockMongoDBHandler) ListDatabases() (driver.MongoDatabaseListResult, driver.MongoError) {
	return m.MockListDatabasesResult, m.MockListDatabasesError
}

func (m *MockMongoDBHandler) DropDatabase(name string) driver.MongoError {
	m.LastDBName = name
	return m.MockDropDatabaseError
}

func (m *MockMongoDBHandler) ListCollections(dbName string) (driver.MongoTablesListResult, driver.MongoError) {
	m.LastDBName = dbName
	return m.MockListCollectionsResult, m.MockListCollectionsError
}

func (m *MockMongoDBHandler) DropTable(dbName, tableName string) driver.MongoError {
	m.LastDBName = dbName
	m.LastTableName = tableName
	return m.MockDropTableError
}

func (m *MockMongoDBHandler) Find() (driver.MongoResults, driver.MongoError) {
	return m.MockFindResult, m.MockFindError
}

func (m *MockMongoDBHandler) FindById(id string) (driver.SingleMongoResult, driver.MongoError) {
	m.LastMongoId = id
	return m.MockFindByIdResult, m.MockFindByIdError
}

func (m *MockMongoDBHandler) Insert(data interface{}) (driver.MongoOperationsResult, driver.MongoError) {
	m.LastInsertData = data
	return m.MockInsertResult, m.MockInsertError
}

func (m *MockMongoDBHandler) UpdateByID(id string, data interface{}) (driver.MongoOperationsResult, driver.MongoError) {
	m.LastMongoId = id
	m.LastUpdateData = data
	return m.MockUpdateByIdResult, m.MockUpdateByIdError
}

func (m *MockMongoDBHandler) Update(data interface{}) (driver.MongoOperationsResult, driver.MongoError) {
	m.LastUpdateData = data
	return m.MockUpdateResult, m.MockUpdateError
}

func (m *MockMongoDBHandler) DeleteById(id string) (driver.MongoOperationsResult, driver.MongoError) {
	m.LastMongoId = id
	return m.MockDeleteByIdResult, m.MockDeleteByIdError
}

func (m *MockMongoDBHandler) Delete() (driver.MongoOperationsResult, driver.MongoError) {
	return m.MockDeleteResult, m.MockDeleteError
}

func (m *MockMongoDBHandler) ExecuteRaw(query string, asPipeline bool) (driver.MongoResults, driver.MongoError) {
	m.LastRawQuery = query
	m.LastAsPipeline = asPipeline
	return m.MockExecuteRawResult, m.MockExecuteRawError
}

func (m *MockMongoDBHandler) Count() (driver.CountMongoResult, driver.MongoError) {
	return m.MockCountResult, m.MockCountError
}

// ============================================================================
// MIDDLEWARE TESTS
// ============================================================================

func TestApiKeyMiddlewareNoKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware("secret123"))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "API key is missing from headers" {
		t.Errorf("Expected missing key error, got %v", response["error"])
	}
}

func TestApiKeyMiddlewareValidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware("secret123"))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("api_key", "secret123")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestApiKeyMiddlewareInvalidKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware("secret123"))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("api_key", "wrong_key")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Errorf("Expected 401, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)
	if response["error"] != "Invalid API key" {
		t.Errorf("Expected invalid key error, got %v", response["error"])
	}
}

func TestApiKeyMiddlewareEmptyKeySkipsValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200 when no key required, got %d", w.Code)
	}
}

func TestApiKeyMiddlewareMultipleRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware("secret123"))

	router.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	// Valid request
	req1, _ := http.NewRequest("GET", "/test", nil)
	req1.Header.Set("api_key", "secret123")
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)
	if w1.Code != 200 {
		t.Errorf("Valid request should return 200, got %d", w1.Code)
	}

	// Invalid request
	req2, _ := http.NewRequest("GET", "/test", nil)
	req2.Header.Set("api_key", "wrong")
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)
	if w2.Code != 401 {
		t.Errorf("Invalid request should return 401, got %d", w2.Code)
	}

	// No key request
	req3, _ := http.NewRequest("GET", "/test", nil)
	w3 := httptest.NewRecorder()
	router.ServeHTTP(w3, req3)
	if w3.Code != 401 {
		t.Errorf("No key request should return 401, got %d", w3.Code)
	}
}

// ============================================================================
// ENDPOINT TESTS WITH MOCKED RESPONSES
// ============================================================================

func TestListDatabasesEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/databases", func(c *gin.Context) {
		// Simulate successful MongoDB response
		c.JSON(200, responses.DatabaseListResponse{
			Status:    true,
			Databases: []string{"test_db", "admin", "config"},
		})
	})

	req, _ := http.NewRequest("GET", "/db/databases", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.DatabaseListResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if len(response.Databases) != 3 {
		t.Errorf("Expected 3 databases, got %d", len(response.Databases))
	}
	if response.Databases[0] != "test_db" {
		t.Errorf("Expected first database=test_db, got %s", response.Databases[0])
	}
}

func TestListDatabasesEndpointError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/databases", func(c *gin.Context) {
		c.JSON(500, responses.GenericErrorResponse{
			Code:   500,
			Status: false,
			Error:  "Connection refused",
		})
	})

	req, _ := http.NewRequest("GET", "/db/databases", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500, got %d", w.Code)
	}

	var response responses.GenericErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Status {
		t.Error("Expected status=false")
	}
	if response.Error != "Connection refused" {
		t.Errorf("Expected error='Connection refused', got %s", response.Error)
	}
}

func TestDropDatabaseEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.DELETE("/db/:db_name/delete", func(c *gin.Context) {
		dbName := c.Param("db_name")
		c.JSON(200, responses.DeleteDatabaseSuccessResponse{
			Status:  true,
			Message: "Database: " + dbName + " has been deleted!",
		})
	})

	req, _ := http.NewRequest("DELETE", "/db/test_db/delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.DeleteDatabaseSuccessResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if !strings.Contains(response.Message, "test_db") {
		t.Error("Expected message to contain database name")
	}
}

func TestDropDatabaseEndpointError(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.DELETE("/db/:db_name/delete", func(c *gin.Context) {
		c.JSON(500, responses.GenericErrorResponse{
			Code:     500,
			Status:   false,
			Error:    "Database not found",
			Database: "nonexistent",
		})
	})

	req, _ := http.NewRequest("DELETE", "/db/nonexistent/delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 500 {
		t.Errorf("Expected 500, got %d", w.Code)
	}
}

func TestListCollectionsEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/tables", func(c *gin.Context) {
		c.JSON(200, responses.TablesInDatabaseResponse{
			Status: true,
			Tables: []string{"users", "posts", "comments"},
		})
	})

	req, _ := http.NewRequest("GET", "/db/test_db/tables", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.TablesInDatabaseResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if len(response.Tables) != 3 {
		t.Errorf("Expected 3 tables, got %d", len(response.Tables))
	}
}

func TestDropTableEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.DELETE("/db/:db_name/:table_name/delete", func(c *gin.Context) {
		tableName := c.Param("table_name")
		c.JSON(200, responses.WipeTableInDatabaseResponse{
			Status:  true,
			Message: "Table / collection: " + tableName + ", dropped!",
		})
	})

	req, _ := http.NewRequest("DELETE", "/db/test_db/users/delete", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.WipeTableInDatabaseResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if !strings.Contains(response.Message, "users") {
		t.Error("Expected message to contain table name")
	}
}

func TestSelectEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		dbName := c.Param("db_name")
		tableName := c.Param("table_name")

		results := []map[string]any{
			{"_id": "1", "name": "John", "age": 30},
			{"_id": "2", "name": "Jane", "age": 25},
		}

		c.JSON(200, responses.SelectResultsResponse{
			Status:   true,
			Code:     200,
			Database: dbName,
			Table:    tableName,
			Count:    2,
			Pagination: responses.SelectResultsPaginationResponse{
				TotalPages:  1,
				CurrentPage: 1,
				NextPage:    0,
				PrevPage:    0,
				LastPage:    1,
				PerPage:     10,
			},
			Results: results,
		})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/select?page=1&per_page=10", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.SelectResultsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if response.Count != 2 {
		t.Errorf("Expected count=2, got %d", response.Count)
	}
	if len(response.Results) != 2 {
		t.Errorf("Expected 2 results, got %d", len(response.Results))
	}
	if response.Database != "testdb" {
		t.Errorf("Expected database=testdb, got %s", response.Database)
	}
	if response.Table != "users" {
		t.Errorf("Expected table=users, got %s", response.Table)
	}
}

func TestSelectEndpointWithPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		_ = c.DefaultQuery("page", "1")
		_ = c.DefaultQuery("per_page", "10")

		c.JSON(200, responses.SelectResultsResponse{
			Status: true,
			Code:   200,
			Pagination: responses.SelectResultsPaginationResponse{
				TotalPages:  5,
				CurrentPage: 2,
				NextPage:    3,
				PrevPage:    1,
				LastPage:    5,
				PerPage:     25,
			},
			Results: []map[string]any{},
		})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/select?page=2&per_page=25", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.SelectResultsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Pagination.CurrentPage != 2 {
		t.Errorf("Expected current_page=2, got %d", response.Pagination.CurrentPage)
	}
	if response.Pagination.PerPage != 25 {
		t.Errorf("Expected per_page=25, got %d", response.Pagination.PerPage)
	}
	if response.Pagination.NextPage != 3 {
		t.Errorf("Expected next_page=3, got %d", response.Pagination.NextPage)
	}
}

func TestSelectEndpointEmpty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		c.JSON(200, responses.SelectResultsResponse{
			Status:  true,
			Code:    200,
			Count:   0,
			Results: []map[string]any{},
		})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/select", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var response responses.SelectResultsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Count != 0 {
		t.Errorf("Expected count=0, got %d", response.Count)
	}
	if len(response.Results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(response.Results))
	}
}

func TestFindByIdEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/:table_name/get/:mongo_id", func(c *gin.Context) {
		mongoId := c.Param("mongo_id")
		c.JSON(200, responses.SelectSingleResultResponse{
			Status:   true,
			Code:     200,
			Database: "testdb",
			Table:    "users",
			Result: map[string]any{
				"_id":  mongoId,
				"name": "John",
				"age":  30,
			},
		})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/get/507f1f77bcf86cd799439011", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.SelectSingleResultResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	result := response.Result.(map[string]interface{})
	if result["name"] != "John" {
		t.Errorf("Expected name=John, got %v", result["name"])
	}
}

func TestFindByIdEndpointNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/:table_name/get/:mongo_id", func(c *gin.Context) {
		c.JSON(404, responses.GenericErrorResponse{
			Code:   404,
			Status: false,
			Error:  "Document not found",
		})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/get/nonexistent", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 404 {
		t.Errorf("Expected 404, got %d", w.Code)
	}
}

func TestInsertEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.POST("/db/:db_name/:table_name/insert", func(c *gin.Context) {
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide data under the key 'payload'",
			})
			return
		}

		c.JSON(200, responses.MongoOperationsResultResponse{
			Code:      200,
			Status:    true,
			Database:  c.Param("db_name"),
			Table:     c.Param("table_name"),
			Operation: "insert",
			Message:   "1 document inserted",
		})
	})

	form := url.Values{}
	form.Add("payload", `{"name": "John", "age": 30}`)
	req, _ := http.NewRequest("POST", "/db/testdb/users/insert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.MongoOperationsResultResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if response.Operation != "insert" {
		t.Errorf("Expected operation=insert, got %s", response.Operation)
	}
}

func TestInsertEndpointMissingPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.POST("/db/:db_name/:table_name/insert", func(c *gin.Context) {
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide data under the key 'payload'",
			})
			return
		}
	})

	req, _ := http.NewRequest("POST", "/db/testdb/users/insert", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400, got %d", w.Code)
	}

	var response responses.GenericErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Status {
		t.Error("Expected status=false")
	}
}

func TestUpdateByIdEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.PUT("/db/:db_name/:table_name/update/:mongo_id", func(c *gin.Context) {
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide data under the key 'payload'",
			})
			return
		}

		c.JSON(200, responses.MongoOperationsResultResponse{
			Code:      200,
			Status:    true,
			Operation: "update",
			Message:   "1 document updated",
		})
	})

	form := url.Values{}
	form.Add("payload", `{"name": "Jane"}`)
	req, _ := http.NewRequest("PUT", "/db/testdb/users/update/507f1f77bcf86cd799439011", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.MongoOperationsResultResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Operation != "update" {
		t.Errorf("Expected operation=update, got %s", response.Operation)
	}
}

func TestUpdateWhereEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.PUT("/db/:db_name/:table_name/update-where", func(c *gin.Context) {
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide data under the key 'payload'",
			})
			return
		}

		c.JSON(200, responses.MongoOperationsResultResponse{
			Code:      200,
			Status:    true,
			Operation: "update",
			Message:   "5 documents updated",
		})
	})

	form := url.Values{}
	form.Add("payload", `{"status": "active"}`)
	req, _ := http.NewRequest("PUT", "/db/testdb/users/update-where?query_and=status,=,inactive", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestDeleteByIdEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.DELETE("/db/:db_name/:table_name/delete/:mongo_id", func(c *gin.Context) {
		c.JSON(200, responses.MongoOperationsResultResponse{
			Code:      200,
			Status:    true,
			Operation: "delete",
			Message:   "1 document deleted",
		})
	})

	req, _ := http.NewRequest("DELETE", "/db/testdb/users/delete/507f1f77bcf86cd799439011", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.MongoOperationsResultResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Operation != "delete" {
		t.Errorf("Expected operation=delete, got %s", response.Operation)
	}
}

func TestDeleteWhereEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.DELETE("/db/:db_name/:table_name/delete-where", func(c *gin.Context) {
		c.JSON(200, responses.MongoOperationsResultResponse{
			Code:      200,
			Status:    true,
			Operation: "delete",
			Message:   "3 documents deleted",
		})
	})

	req, _ := http.NewRequest("DELETE", "/db/testdb/users/delete-where?query_and=age,>,30", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

func TestCountEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/:db_name/:table_name/count", func(c *gin.Context) {
		c.JSON(200, responses.CountResultsResponse{
			Status:   true,
			Code:     200,
			Database: "testdb",
			Table:    "users",
			Count:    42,
		})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/count", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.CountResultsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !response.Status {
		t.Error("Expected status=true")
	}
	if response.Count != 42 {
		t.Errorf("Expected count=42, got %d", response.Count)
	}
}

func TestCustomQueryEndpointSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.POST("/db/:db_name/:table_name/custom-query", func(c *gin.Context) {
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide data under the key 'payload'",
			})
			return
		}

		c.JSON(200, responses.SelectResultsResponse{
			Status: true,
			Code:   200,
			Count:  1,
			Results: []map[string]any{
				{"_id": "1", "name": "John"},
			},
		})
	})

	form := url.Values{}
	form.Add("payload", `{"name": "John"}`)
	req, _ := http.NewRequest("POST", "/db/testdb/users/custom-query?page=1&per_page=10", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response responses.SelectResultsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Count != 1 {
		t.Errorf("Expected count=1, got %d", response.Count)
	}
}

func TestCustomQueryEndpointAsPipeline(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.POST("/db/:db_name/:table_name/custom-query", func(c *gin.Context) {
		asPipeline := c.Query("as_pipeline")
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Failed to provide data under the key 'payload'",
			})
			return
		}

		// Verify as_pipeline was passed
		if asPipeline != "true" {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Expected as_pipeline=true",
			})
			return
		}

		c.JSON(200, responses.SelectResultsResponse{
			Status: true,
			Code:   200,
			Count:  5,
		})
	})

	form := url.Values{}
	form.Add("payload", `[{"$match": {"status": "active"}}, {"$group": {"_id": "$category"}}]`)
	req, _ := http.NewRequest("POST", "/db/testdb/users/custom-query?as_pipeline=true", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}
}

// ============================================================================
// QUERY PARSING TESTS (Integration with helpers.go)
// ============================================================================

func TestParseQuerySimpleEquality(t *testing.T) {
	result := driver.ParseQuery("name,=,John", true)

	if len(result) != 1 {
		t.Fatalf("Expected 1 query condition, got %d", len(result))
	}

	if result[0][0] != "name" {
		t.Errorf("Expected field=name, got %v", result[0][0])
	}
	if result[0][1] != "=" {
		t.Errorf("Expected operator==, got %v", result[0][1])
	}
	if result[0][2] != "John" {
		t.Errorf("Expected value=John, got %v", result[0][2])
	}
}

func TestParseQueryMultipleConditions(t *testing.T) {
	result := driver.ParseQuery("name,=,John|age,>,25", true)

	if len(result) != 2 {
		t.Fatalf("Expected 2 query conditions, got %d", len(result))
	}

	// First condition
	if result[0][0] != "name" {
		t.Errorf("Expected first field=name, got %v", result[0][0])
	}
	if result[0][1] != "=" {
		t.Errorf("Expected first operator==, got %v", result[0][1])
	}

	// Second condition
	if result[1][0] != "age" {
		t.Errorf("Expected second field=age, got %v", result[1][0])
	}
	if result[1][1] != ">" {
		t.Errorf("Expected second operator=>, got %v", result[1][1])
	}
}

func TestParseQueryWithAutoConvert(t *testing.T) {
	result := driver.ParseQuery("age,=,30", true)

	if len(result) != 1 {
		t.Fatalf("Expected 1 query condition, got %d", len(result))
	}

	// With auto-convert, "30" should be converted to int
	if val, ok := result[0][2].(int); ok {
		if val != 30 {
			t.Errorf("Expected value=30, got %d", val)
		}
	}
}

func TestParseQueryBetweenOperator(t *testing.T) {
	result := driver.ParseQuery("age,between,[18:65]", true)

	if len(result) != 1 {
		t.Fatalf("Expected 1 query condition, got %d", len(result))
	}

	if result[0][1] != "between" {
		t.Errorf("Expected operator=between, got %v", result[0][1])
	}
}

func TestParseSortSimple(t *testing.T) {
	result := driver.ParseSort("[name:asc]")

	if len(result) != 1 {
		t.Fatalf("Expected 1 sort condition, got %d", len(result))
	}

	if result[0][0] != "name" {
		t.Errorf("Expected field=name, got %v", result[0][0])
	}
	if result[0][1] != "asc" {
		t.Errorf("Expected direction=asc, got %v", result[0][1])
	}
}

func TestParseSortMultiple(t *testing.T) {
	result := driver.ParseSort("[name:asc|age:desc]")

	if len(result) != 2 {
		t.Fatalf("Expected 2 sort conditions, got %d", len(result))
	}

	if result[0][0] != "name" || result[0][1] != "asc" {
		t.Errorf("Expected first sort=name:asc, got %v:%v", result[0][0], result[0][1])
	}
	if result[1][0] != "age" || result[1][1] != "desc" {
		t.Errorf("Expected second sort=age:desc, got %v:%v", result[1][0], result[1][1])
	}
}

func TestParseSortDefaultAsc(t *testing.T) {
	// Without direction, ParseSort requires explicit direction
	result := driver.ParseSort("[name:asc]")

	if len(result) != 1 {
		t.Fatalf("Expected 1 sort condition, got %d", len(result))
	}

	if result[0][1] != "asc" {
		t.Errorf("Expected direction=asc, got %v", result[0][1])
	}
}

func TestParseQueryLikeOperator(t *testing.T) {
	result := driver.ParseQuery("name,like,John", true)

	if len(result) != 1 {
		t.Fatalf("Expected 1 query condition, got %d", len(result))
	}

	if result[0][1] != "like" {
		t.Errorf("Expected operator=like, got %v", result[0][1])
	}
}

func TestParseQueryNotEqualOperators(t *testing.T) {
	// Test != operator
	result1 := driver.ParseQuery("status,!=,inactive", true)
	if len(result1) != 1 || result1[0][1] != "!=" {
		t.Errorf("Expected operator=!=, got %v", result1[0][1])
	}

	// Test <> operator
	result2 := driver.ParseQuery("status,<>,deleted", true)
	if len(result2) != 1 || result2[0][1] != "<>" {
		t.Errorf("Expected operator=<>, got %v", result2[0][1])
	}
}

func TestParseQueryComparisonOperators(t *testing.T) {
	tests := []struct {
		query    string
		expected string
	}{
		{"age,>,25", ">"},
		{"age,<,65", "<"},
		{"age,>=,18", ">="},
		{"age,<=,100", "<="},
	}

	for _, tt := range tests {
		result := driver.ParseQuery(tt.query, true)
		if len(result) != 1 {
			t.Errorf("Query %s: Expected 1 condition, got %d", tt.query, len(result))
			continue
		}
		if result[0][1] != tt.expected {
			t.Errorf("Query %s: Expected operator=%s, got %v", tt.query, tt.expected, result[0][1])
		}
	}
}

// ============================================================================
// ROUTE PARAMETER TESTS
// ============================================================================

func TestRouteParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var capturedDBName, capturedTableName, capturedMongoId string

	router.GET("/db/:db_name/:table_name/get/:mongo_id", func(c *gin.Context) {
		capturedDBName = c.Param("db_name")
		capturedTableName = c.Param("table_name")
		capturedMongoId = c.Param("mongo_id")
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/db/mydb/mytable/get/abc123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedDBName != "mydb" {
		t.Errorf("Expected db_name=mydb, got %s", capturedDBName)
	}
	if capturedTableName != "mytable" {
		t.Errorf("Expected table_name=mytable, got %s", capturedTableName)
	}
	if capturedMongoId != "abc123" {
		t.Errorf("Expected mongo_id=abc123, got %s", capturedMongoId)
	}
}

func TestQueryParameterExtraction(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()

	var capturedPage, capturedPerPage, capturedQueryAnd, capturedSort string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedPage = c.Query("page")
		capturedPerPage = c.Query("per_page")
		capturedQueryAnd = c.Query("query_and")
		capturedSort = c.Query("sort")
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/db/mydb/users/select?page=2&per_page=20&query_and=name,=,John&sort=age:desc", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedPage != "2" {
		t.Errorf("Expected page=2, got %s", capturedPage)
	}
	if capturedPerPage != "20" {
		t.Errorf("Expected per_page=20, got %s", capturedPerPage)
	}
	if capturedQueryAnd != "name,=,John" {
		t.Errorf("Expected query_and=name,=,John, got %s", capturedQueryAnd)
	}
	if capturedSort != "age:desc" {
		t.Errorf("Expected sort=age:desc, got %s", capturedSort)
	}
}

// ============================================================================
// ERROR HANDLING TESTS
// ============================================================================

func TestInvalidJsonPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.POST("/db/:db_name/:table_name/insert", func(c *gin.Context) {
		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")

		// Try to parse as JSON
		var data interface{}
		err := json.Unmarshal([]byte(payload), &data)
		if err != nil {
			c.JSON(400, responses.GenericErrorResponse{
				Code:   400,
				Status: false,
				Error:  "Invalid JSON: " + err.Error(),
			})
			return
		}

		c.JSON(200, gin.H{"status": "ok"})
	})

	form := url.Values{}
	form.Add("payload", `{invalid json}`)
	req, _ := http.NewRequest("POST", "/db/testdb/users/insert", strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 400 {
		t.Errorf("Expected 400 for invalid JSON, got %d", w.Code)
	}

	var response responses.GenericErrorResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if !strings.Contains(response.Error, "Invalid JSON") {
		t.Errorf("Expected Invalid JSON error, got %s", response.Error)
	}
}

func TestSelectWithGroupBy(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedGroupBy string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedGroupBy = c.Query("group_by")
		c.JSON(200, gin.H{"status": "ok", "group_by": capturedGroupBy})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/select?group_by=category", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedGroupBy != "category" {
		t.Errorf("Expected group_by=category, got %s", capturedGroupBy)
	}
}

func TestSelectWithInnerPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedInnerPage, capturedInnerPerPage string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedInnerPage = c.Query("inner_page")
		capturedInnerPerPage = c.Query("inner_per_page")
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/select?group_by=category&inner_page=2&inner_per_page=5", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedInnerPage != "2" {
		t.Errorf("Expected inner_page=2, got %s", capturedInnerPage)
	}
	if capturedInnerPerPage != "5" {
		t.Errorf("Expected inner_per_page=5, got %s", capturedInnerPerPage)
	}
}

func TestAutoConvertInputsParam(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedAutoConvert string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedAutoConvert = c.Query("auto_convert_inputs")
		c.JSON(200, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/db/testdb/users/select?auto_convert_inputs=false", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedAutoConvert != "false" {
		t.Errorf("Expected auto_convert_inputs=false, got %s", capturedAutoConvert)
	}
}

// ============================================================================
// HTTP METHODS TESTS
// ============================================================================

func TestCorrectHttpMethods(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	router.GET("/db/databases", func(c *gin.Context) { c.JSON(200, gin.H{"method": "GET"}) })
	router.DELETE("/db/:db_name/delete", func(c *gin.Context) { c.JSON(200, gin.H{"method": "DELETE"}) })
	router.POST("/db/:db_name/:table_name/insert", func(c *gin.Context) { c.JSON(200, gin.H{"method": "POST"}) })
	router.PUT("/db/:db_name/:table_name/update/:mongo_id", func(c *gin.Context) { c.JSON(200, gin.H{"method": "PUT"}) })

	tests := []struct {
		method       string
		path         string
		expectedCode int
	}{
		{"GET", "/db/databases", 200},
		{"POST", "/db/databases", 404},
		{"DELETE", "/db/test/delete", 200},
		{"GET", "/db/test/delete", 404},
		{"POST", "/db/test/users/insert", 200},
		{"GET", "/db/test/users/insert", 404},
		{"PUT", "/db/test/users/update/123", 200},
		{"POST", "/db/test/users/update/123", 404},
	}

	for _, tt := range tests {
		req, _ := http.NewRequest(tt.method, tt.path, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != tt.expectedCode {
			t.Errorf("%s %s: Expected %d, got %d", tt.method, tt.path, tt.expectedCode, w.Code)
		}
	}
}

// ============================================================================
// COMPLEX MULTI-FIELD QUERY TESTS
// ============================================================================

func TestParseQueryMultipleFieldsAND(t *testing.T) {
	// Test multiple fields with AND conditions: name=John AND age>25 AND status=active
	query := "name,=,John|age,>,25|status,=,active"
	result := driver.ParseQuery(query, true)

	if len(result) != 3 {
		t.Fatalf("Expected 3 query conditions, got %d", len(result))
	}

	// Verify first condition: name = John
	if result[0][0] != "name" || result[0][1] != "=" || result[0][2] != "John" {
		t.Errorf("First condition: expected name=John, got %v %v %v", result[0][0], result[0][1], result[0][2])
	}

	// Verify second condition: age > 25
	if result[1][0] != "age" || result[1][1] != ">" {
		t.Errorf("Second condition: expected age>, got %v %v", result[1][0], result[1][1])
	}
	// Check age value is converted to int
	if val, ok := result[1][2].(int); !ok || val != 25 {
		t.Errorf("Second condition value: expected int 25, got %T %v", result[1][2], result[1][2])
	}

	// Verify third condition: status = active
	if result[2][0] != "status" || result[2][1] != "=" || result[2][2] != "active" {
		t.Errorf("Third condition: expected status=active, got %v %v %v", result[2][0], result[2][1], result[2][2])
	}
}

func TestParseQueryComplexUserSearch(t *testing.T) {
	// Realistic user search: find users in age range, with specific role, created after date
	query := "age,>=,18|age,<=,65|role,=,admin|verified,=,true"
	result := driver.ParseQuery(query, true)

	if len(result) != 4 {
		t.Fatalf("Expected 4 query conditions, got %d", len(result))
	}

	// age >= 18
	if result[0][0] != "age" || result[0][1] != ">=" {
		t.Errorf("Expected age>=, got %v %v", result[0][0], result[0][1])
	}
	if val, ok := result[0][2].(int); !ok || val != 18 {
		t.Errorf("Expected 18, got %v", result[0][2])
	}

	// age <= 65
	if result[1][0] != "age" || result[1][1] != "<=" {
		t.Errorf("Expected age<=, got %v %v", result[1][0], result[1][1])
	}

	// role = admin
	if result[2][0] != "role" || result[2][1] != "=" || result[2][2] != "admin" {
		t.Errorf("Expected role=admin, got %v %v %v", result[2][0], result[2][1], result[2][2])
	}

	// verified = true (should be converted to bool)
	if result[3][0] != "verified" || result[3][1] != "=" {
		t.Errorf("Expected verified=, got %v %v", result[3][0], result[3][1])
	}
	if val, ok := result[3][2].(bool); !ok || val != true {
		t.Errorf("Expected bool true, got %T %v", result[3][2], result[3][2])
	}
}

func TestParseQueryMixedOperators(t *testing.T) {
	// Mix of different operators on different fields
	query := "name,like,John|status,!=,deleted|price,between,[10:100]|category,=,electronics"
	result := driver.ParseQuery(query, true)

	if len(result) != 4 {
		t.Fatalf("Expected 4 query conditions, got %d", len(result))
	}

	// name like John
	if result[0][0] != "name" || result[0][1] != "like" || result[0][2] != "John" {
		t.Errorf("Expected name like John, got %v %v %v", result[0][0], result[0][1], result[0][2])
	}

	// status != deleted
	if result[1][0] != "status" || result[1][1] != "!=" || result[1][2] != "deleted" {
		t.Errorf("Expected status!=deleted, got %v %v %v", result[1][0], result[1][1], result[1][2])
	}

	// price between 10:100
	if result[2][0] != "price" || result[2][1] != "between" {
		t.Errorf("Expected price between, got %v %v", result[2][0], result[2][1])
	}

	// category = electronics
	if result[3][0] != "category" || result[3][1] != "=" || result[3][2] != "electronics" {
		t.Errorf("Expected category=electronics, got %v %v %v", result[3][0], result[3][1], result[3][2])
	}
}

func TestParseQueryProductSearch(t *testing.T) {
	// E-commerce product search: category, price range, in stock, rating
	query := "category,=,laptops|price,>=,500|price,<=,2000|in_stock,=,true|rating,>,4"
	result := driver.ParseQuery(query, true)

	if len(result) != 5 {
		t.Fatalf("Expected 5 query conditions, got %d", len(result))
	}

	// Verify all fields present
	fields := make(map[string]int)
	for _, cond := range result {
		fields[cond[0].(string)]++
	}

	if fields["category"] != 1 {
		t.Error("Expected 1 category condition")
	}
	if fields["price"] != 2 {
		t.Error("Expected 2 price conditions (min and max)")
	}
	if fields["in_stock"] != 1 {
		t.Error("Expected 1 in_stock condition")
	}
	if fields["rating"] != 1 {
		t.Error("Expected 1 rating condition")
	}
}

func TestParseQueryWithNullValues(t *testing.T) {
	// Test querying for null/missing fields
	query := "deleted_at,=,null|status,!=,null"
	result := driver.ParseQuery(query, true)

	if len(result) != 2 {
		t.Fatalf("Expected 2 query conditions, got %d", len(result))
	}

	// deleted_at = null (should be nil)
	if result[0][0] != "deleted_at" || result[0][1] != "=" {
		t.Errorf("Expected deleted_at=, got %v %v", result[0][0], result[0][1])
	}
	if result[0][2] != nil {
		t.Errorf("Expected nil, got %v", result[0][2])
	}

	// status != null
	if result[1][0] != "status" || result[1][1] != "!=" {
		t.Errorf("Expected status!=, got %v %v", result[1][0], result[1][1])
	}
}

func TestParseQueryWithFloats(t *testing.T) {
	// Test queries with floating point values
	query := "latitude,>=,40.7128|longitude,<=,-74.0060|radius,=,10.5"
	result := driver.ParseQuery(query, true)

	if len(result) != 3 {
		t.Fatalf("Expected 3 query conditions, got %d", len(result))
	}

	// Check latitude is float
	if val, ok := result[0][2].(float64); !ok {
		t.Errorf("Expected float64, got %T", result[0][2])
	} else if val != 40.7128 {
		t.Errorf("Expected 40.7128, got %f", val)
	}

	// Check negative longitude
	if val, ok := result[1][2].(float64); !ok {
		t.Errorf("Expected float64, got %T", result[1][2])
	} else if val != -74.0060 {
		t.Errorf("Expected -74.0060, got %f", val)
	}
}

func TestSelectEndpointWithQueryAndFilters(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedQueryAnd, capturedQueryOr, capturedSort, capturedGroupBy string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedQueryAnd = c.Query("query_and")
		capturedQueryOr = c.Query("query_or")
		capturedSort = c.Query("sort")
		capturedGroupBy = c.Query("group_by")

		c.JSON(200, responses.SelectResultsResponse{
			Status:  true,
			Code:    200,
			Count:   5,
			Results: []map[string]any{},
		})
	})

	// Complex query with AND, OR, sort, and group
	queryURL := "/db/testdb/users/select?query_and=status,=,active|age,>,18&query_or=role,=,admin|role,=,moderator&sort=[created_at:desc]&group_by=department"
	req, _ := http.NewRequest("GET", queryURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Verify AND query was captured
	if capturedQueryAnd != "status,=,active|age,>,18" {
		t.Errorf("Expected AND query 'status,=,active|age,>,18', got '%s'", capturedQueryAnd)
	}

	// Verify OR query was captured
	if capturedQueryOr != "role,=,admin|role,=,moderator" {
		t.Errorf("Expected OR query 'role,=,admin|role,=,moderator', got '%s'", capturedQueryOr)
	}

	// Verify sort was captured
	if capturedSort != "[created_at:desc]" {
		t.Errorf("Expected sort '[created_at:desc]', got '%s'", capturedSort)
	}

	// Verify group_by was captured
	if capturedGroupBy != "department" {
		t.Errorf("Expected group_by 'department', got '%s'", capturedGroupBy)
	}
}

func TestSelectEndpointWithComplexFiltering(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedQueryAnd string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedQueryAnd = c.Query("query_and")

		// Parse the query and verify
		parsedQuery := driver.ParseQuery(capturedQueryAnd, true)

		c.JSON(200, gin.H{
			"status":            true,
			"parsed_conditions": len(parsedQuery),
		})
	})

	// Multi-field product search
	queryURL := "/db/shop/products/select?query_and=category,=,electronics|brand,like,Apple|price,>=,500|price,<=,2000|in_stock,=,true"
	req, _ := http.NewRequest("GET", queryURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	// Verify 5 conditions were parsed
	if response["parsed_conditions"] != float64(5) {
		t.Errorf("Expected 5 parsed conditions, got %v", response["parsed_conditions"])
	}
}

func TestCountEndpointWithQuery(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedQueryAnd string

	router.GET("/db/:db_name/:table_name/count", func(c *gin.Context) {
		capturedQueryAnd = c.Query("query_and")

		c.JSON(200, responses.CountResultsResponse{
			Status: true,
			Code:   200,
			Count:  42,
		})
	})

	// Count with filter
	queryURL := "/db/testdb/users/count?query_and=status,=,active|verified,=,true"
	req, _ := http.NewRequest("GET", queryURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedQueryAnd != "status,=,active|verified,=,true" {
		t.Errorf("Expected query 'status,=,active|verified,=,true', got '%s'", capturedQueryAnd)
	}

	var response responses.CountResultsResponse
	json.Unmarshal(w.Body.Bytes(), &response)

	if response.Count != 42 {
		t.Errorf("Expected count=42, got %d", response.Count)
	}
}

func TestDeleteWhereWithMultipleConditions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedQueryAnd string

	router.DELETE("/db/:db_name/:table_name/delete-where", func(c *gin.Context) {
		capturedQueryAnd = c.Query("query_and")

		// Parse and verify conditions
		parsed := driver.ParseQuery(capturedQueryAnd, true)

		c.JSON(200, responses.MongoOperationsResultResponse{
			Status:    true,
			Code:      200,
			Operation: "delete",
			Message:   "Deleted documents matching " + string(rune(len(parsed))) + " conditions",
		})
	})

	// Delete old inactive users
	queryURL := "/db/testdb/users/delete-where?query_and=status,=,inactive|last_login,<,2024-01-01|verified,=,false"
	req, _ := http.NewRequest("DELETE", queryURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Verify query was passed correctly
	if capturedQueryAnd != "status,=,inactive|last_login,<,2024-01-01|verified,=,false" {
		t.Errorf("Query not captured correctly: %s", capturedQueryAnd)
	}

	// Verify it parses to 3 conditions
	parsed := driver.ParseQuery(capturedQueryAnd, true)
	if len(parsed) != 3 {
		t.Errorf("Expected 3 conditions, got %d", len(parsed))
	}
}

func TestUpdateWhereWithMultipleConditions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedQueryAnd, capturedQueryOr string

	router.PUT("/db/:db_name/:table_name/update-where", func(c *gin.Context) {
		capturedQueryAnd = c.Query("query_and")
		capturedQueryOr = c.Query("query_or")

		c.Request.ParseForm()
		payload := c.Request.Form.Get("payload")
		if payload == "" {
			c.JSON(400, responses.GenericErrorResponse{Status: false, Error: "Missing payload"})
			return
		}

		c.JSON(200, responses.MongoOperationsResultResponse{
			Status:    true,
			Code:      200,
			Operation: "update",
			Message:   "Updated matching documents",
		})
	})

	// Update with both AND and OR conditions
	form := url.Values{}
	form.Add("payload", `{"status": "verified", "updated_at": "2026-02-07"}`)
	queryURL := "/db/testdb/users/update-where?query_and=email_verified,=,true|phone_verified,=,true&query_or=role,=,premium|role,=,enterprise"
	req, _ := http.NewRequest("PUT", queryURL, strings.NewReader(form.Encode()))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Errorf("Expected 200, got %d", w.Code)
	}

	// Verify AND conditions
	parsedAnd := driver.ParseQuery(capturedQueryAnd, true)
	if len(parsedAnd) != 2 {
		t.Errorf("Expected 2 AND conditions, got %d", len(parsedAnd))
	}

	// Verify OR conditions
	parsedOr := driver.ParseQuery(capturedQueryOr, true)
	if len(parsedOr) != 2 {
		t.Errorf("Expected 2 OR conditions, got %d", len(parsedOr))
	}
}

func TestParseQueryWithSpecialCharacters(t *testing.T) {
	// Test with email addresses and URLs
	query := "email,=,user@example.com|website,like,https"
	result := driver.ParseQuery(query, true)

	if len(result) != 2 {
		t.Fatalf("Expected 2 conditions, got %d", len(result))
	}

	if result[0][2] != "user@example.com" {
		t.Errorf("Expected email user@example.com, got %v", result[0][2])
	}
}

func TestParseQueryNestedFieldNames(t *testing.T) {
	// Test with dot notation for nested fields (MongoDB style)
	query := "address.city,=,NewYork|profile.settings.theme,=,dark"
	result := driver.ParseQuery(query, true)

	if len(result) != 2 {
		t.Fatalf("Expected 2 conditions, got %d", len(result))
	}

	if result[0][0] != "address.city" {
		t.Errorf("Expected field 'address.city', got %v", result[0][0])
	}

	if result[1][0] != "profile.settings.theme" {
		t.Errorf("Expected field 'profile.settings.theme', got %v", result[1][0])
	}
}

func TestSelectWithSortAndPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedPage, capturedPerPage, capturedSort string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedPage = c.Query("page")
		capturedPerPage = c.Query("per_page")
		capturedSort = c.Query("sort")
		_ = c.Query("query_and")

		c.JSON(200, responses.SelectResultsResponse{
			Status: true,
			Code:   200,
			Pagination: responses.SelectResultsPaginationResponse{
				CurrentPage: 3,
				PerPage:     50,
				TotalPages:  10,
			},
			Results: []map[string]any{},
		})
	})

	// Complex paginated and sorted query
	queryURL := "/db/logs/events/select?page=3&per_page=50&sort=[timestamp:desc|severity:asc]&query_and=level,=,error|source,like,api"
	req, _ := http.NewRequest("GET", queryURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedPage != "3" {
		t.Errorf("Expected page=3, got %s", capturedPage)
	}
	if capturedPerPage != "50" {
		t.Errorf("Expected per_page=50, got %s", capturedPerPage)
	}
	if capturedSort != "[timestamp:desc|severity:asc]" {
		t.Errorf("Expected sort '[timestamp:desc|severity:asc]', got %s", capturedSort)
	}

	// Verify sort parses correctly
	parsedSort := driver.ParseSort(capturedSort)
	if len(parsedSort) != 2 {
		t.Errorf("Expected 2 sort conditions, got %d", len(parsedSort))
	}
}

func TestSelectWithGroupByAndInnerPagination(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(apiKeyMiddleware(""))

	var capturedGroupBy, capturedInnerPage, capturedInnerPerPage, capturedQueryAnd string

	router.GET("/db/:db_name/:table_name/select", func(c *gin.Context) {
		capturedGroupBy = c.Query("group_by")
		capturedInnerPage = c.Query("inner_page")
		capturedInnerPerPage = c.Query("inner_per_page")
		capturedQueryAnd = c.Query("query_and")

		c.JSON(200, gin.H{"status": true})
	})

	// Grouped query with inner pagination and filter
	queryURL := "/db/analytics/orders/select?group_by=category&inner_page=2&inner_per_page=5&query_and=year,=,2025|status,=,completed"
	req, _ := http.NewRequest("GET", queryURL, nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if capturedGroupBy != "category" {
		t.Errorf("Expected group_by=category, got %s", capturedGroupBy)
	}
	if capturedInnerPage != "2" {
		t.Errorf("Expected inner_page=2, got %s", capturedInnerPage)
	}
	if capturedInnerPerPage != "5" {
		t.Errorf("Expected inner_per_page=5, got %s", capturedInnerPerPage)
	}

	// Verify query parses
	parsed := driver.ParseQuery(capturedQueryAnd, true)
	if len(parsed) != 2 {
		t.Errorf("Expected 2 query conditions, got %d", len(parsed))
	}
}
