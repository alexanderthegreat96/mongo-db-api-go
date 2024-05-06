package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type AggregationStage struct {
	StageName string
	Params    interface{}
}
type MongoDBHandler struct {
	debug               bool
	limit               int
	perPage             int
	page                int
	sort                []primitive.E
	query               map[string]interface{}
	aggregationPipeline []AggregationStage
	multipleWheres      bool
	host                string
	port                string
	dbName              string
	tableName           string
	username            string
	password            string
	useTimestamps       bool
	timeNow             time.Time
	client              *mongo.Client
	db                  *mongo.Database
	collection          *mongo.Collection
}

// used for results mapping
type MongoResultPagination struct {
	TotalPages  int
	CurrentPage int
	NextPage    int
	LastPage    int
	PerPage     int
}

// results mapping
type MongoResults struct {
	Status     bool
	Code       int
	Database   string
	Table      string
	Count      int64
	Results    []map[string]interface{}
	Pagination MongoResultPagination
	Query      string
}

// results error
type MongoError struct {
	Status   bool
	Code     int
	Database string
	Table    string
	Error    string
	Query    string
}

type MongoOperationsResult struct {
	Status    bool
	Code      int
	Database  string
	Table     string
	Operation string
	Message   string
	Query     string
}

// initializes the mongo handler driver
// similar to how constructors work
func MongoDB() *MongoDBHandler {
	host := helpers.GetEnv("MONGO_DB_HOST", "localhost")
	port := helpers.GetEnv("MONGO_DB_PORT", "27017")
	dbName := helpers.GetEnv("MONGO_DB_NAME=", "test")
	tableName := helpers.GetEnv("MONGO_DB_TABLE", "test")
	username := helpers.GetEnv("MONGO_DB_USERNAME", "admin")
	password := helpers.GetEnv("MONGO_DB_PASSWORD", "admin")
	useTimestamps, _ := strconv.ParseBool(helpers.GetEnv("HANDLER_USE_TIMESTAMPS", "true"))
	debug, _ := strconv.ParseBool(helpers.GetEnv("HANDLER_DEBUG", "false"))

	return &MongoDBHandler{
		debug:          debug,
		limit:          0,
		perPage:        10,
		page:           1,
		sort:           []primitive.E{},
		query:          make(map[string]interface{}),
		multipleWheres: false,
		host:           host,
		port:           port,
		dbName:         dbName,
		tableName:      tableName,
		username:       username,
		password:       password,
		useTimestamps:  useTimestamps,
		timeNow:        time.Now(),
		client:         nil,
		db:             nil,
		collection:     nil,
	}
}

// handles connectivity
func (mh *MongoDBHandler) getConnection() MongoError {
	if mh.client != nil {
		return mh.newMongoError(500, "Client was empty")
	}

	clientOptions := options.Client().ApplyURI("mongodb://" + mh.host + ":" + mh.port).
		SetAuth(options.Credential{
			Username: mh.username,
			Password: mh.password,
		}).
		SetSocketTimeout(2 * time.Second)

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return mh.newMongoError(500, err.Error())
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return mh.newMongoError(500, err.Error())
	}

	mh.client = client
	mh.db = mh.client.Database(mh.dbName)
	mh.collection = mh.db.Collection(mh.tableName)

	return MongoError{}
}

// set db name
func (mh *MongoDBHandler) DB(dbName string) *MongoDBHandler {
	mh.dbName = dbName
	return mh
}

// set table / collection name
func (mh *MongoDBHandler) Table(tableName string) *MongoDBHandler {
	mh.tableName = tableName
	return mh
}

// provide page if possible
func (mh *MongoDBHandler) Page(page int) *MongoDBHandler {
	mh.page = page
	return mh
}

// provide the limit of records per page
func (mh *MongoDBHandler) PerPage(perPage int) *MongoDBHandler {
	mh.perPage = perPage
	return mh
}

// Where contraint
func (mh *MongoDBHandler) Where(field string, operator string, value interface{}) *MongoDBHandler {
	if field == "_id" {
		objectID, err := primitive.ObjectIDFromHex(value.(string))
		if err != nil {
			return mh
		}
		value = objectID
	}

	mappedValue, err := mapOperators(operator, value)
	if err != nil {
		fmt.Println("Error:", err)
		return mh
	}

	// basically, merge multiple Where conditions into $and
	// if multiple calls to .Where are placed
	if mh.multipleWheres {
		andCondition := make([]interface{}, 0)
		if mh.query != nil {
			for key, val := range mh.query {
				if key != "$and" {
					andCondition = append(andCondition, map[string]interface{}{key: val})
				}
			}
		}
		andCondition = append(andCondition, map[string]interface{}{field: mappedValue})

		mh.query = map[string]interface{}{
			"$and": andCondition,
		}
	} else {
		mh.query[field] = mappedValue
		mh.multipleWheres = true
	}

	return mh
}

// or where constraint
func (mh *MongoDBHandler) OrWhere(field, operator string, value interface{}) *MongoDBHandler {
	if field == "_id" {
		objectID, err := primitive.ObjectIDFromHex(value.(string))
		if err != nil {
			return mh
		}
		value = objectID
	}
	or_condition := make(map[string]interface{})

	mappedValue, err := mapOperators(operator, value)
	if err != nil {
		fmt.Println("Error:", err)
		return mh
	}
	or_condition[field] = mappedValue

	if _, ok := mh.query["$or"]; !ok {
		mh.query["$or"] = []interface{}{}
	}

	mh.query["$or"] = append(mh.query["$or"].([]interface{}), or_condition)

	return mh
}

// ascending / descending sorting -> 1, -1
func (mh *MongoDBHandler) SortBy(field, order string) *MongoDBHandler {
	if mh.sort == nil {
		mh.sort = []primitive.E{}
	}

	var sortOrder int32
	if order == "ASC" {
		sortOrder = 1
	} else {
		sortOrder = -1
	}

	mh.sort = append(mh.sort, primitive.E{Key: field, Value: sortOrder})
	return mh
}

// Group By
func (mh *MongoDBHandler) GroupBy(field string) *MongoDBHandler {
	// should add sorting data as well
	if len(mh.sort) > 0 {
		sortCriteria := make(bson.D, len(mh.sort))

		for i, element := range mh.sort {
			sortCriteria[i] = bson.E{Key: element.Key, Value: element.Value}
		}

		sortStage := bson.D{
			{Key: "$sort", Value: sortCriteria},
		}

		mh.aggregationPipeline = append(mh.aggregationPipeline, AggregationStage{"$sort", sortStage})
	}

	// implement $match
	// $match should contain anything that is handled by
	// query basically

	groupStage := bson.D{
		{Key: "$group", Value: bson.D{
			{Key: "_id", Value: "$" + field},
		}},
	}

	mh.aggregationPipeline = append(mh.aggregationPipeline, AggregationStage{"$group", groupStage})

	return mh
}

// setting up results limit
func (mh *MongoDBHandler) Limit(limit int) *MongoDBHandler {
	mh.limit = limit
	return mh
}

// just adds created_at and updated_at timestamps
// upon insertion and or update
func (mh *MongoDBHandler) appendTimestamps(data interface{}) interface{} {
	if !mh.useTimestamps {
		return data
	}

	switch d := data.(type) {
	case []interface{}:
		for _, item := range d {
			if itemMap, ok := item.(map[string]interface{}); ok {
				if _, exists := itemMap["created_at"]; !exists {
					itemMap["created_at"] = mh.timeNow
				}
				if _, exists := itemMap["updated_at"]; !exists {
					itemMap["updated_at"] = mh.timeNow
				}
			}
		}
	case map[string]interface{}:
		if _, exists := d["created_at"]; !exists {
			d["created_at"] = mh.timeNow
		}
		if _, exists := d["updated_at"]; !exists {
			d["updated_at"] = mh.timeNow
		}
	}

	return data
}

// takes as input a data interface
// can provide 1 or more maps inside
// a slice

func (mh *MongoDBHandler) Insert(data interface{}) (MongoOperationsResult, MongoError) {
	if err := mh.getConnection(); err.Status {
		return MongoOperationsResult{}, err
	}

	if mh.useTimestamps {
		switch d := data.(type) {
		case []map[string]interface{}:
			for i := range d {
				d[i] = mh.appendTimestamps(d[i]).(map[string]interface{})
			}
		case map[string]interface{}:
			data = mh.appendTimestamps(d).(map[string]interface{})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch d := data.(type) {
	case []map[string]interface{}:
		var interfaceSlice []interface{}
		for _, item := range d {
			interfaceSlice = append(interfaceSlice, item)
		}
		_, err := mh.collection.InsertMany(ctx, interfaceSlice)
		if err != nil {
			return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
		}
	case map[string]interface{}:
		_, err := mh.collection.InsertOne(ctx, d)
		if err != nil {
			return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
		}
	default:
		return MongoOperationsResult{}, mh.newMongoError(400, "unsuported data type")
	}

	return mh.newMongoOperations(200, true, "insert", "Insert performed."), MongoError{}
}

// deletes a mongo db
func (mh *MongoDBHandler) DropDatabase(dbName string) MongoError {
	if err := mh.getConnection(); err.Status {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mh.client.Database(dbName).Drop(ctx)
	if err != nil {
		return mh.newMongoError(500, err.Error())
	}

	return MongoError{}
}

// deletes a mongo collection / table
func (mh *MongoDBHandler) DropTable(tableName string) MongoError {
	if err := mh.getConnection(); err.Status {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mh.collection.Drop(ctx)
	if err != nil {
		return mh.newMongoError(500, err.Error())
	}

	return MongoError{}
}

func (mh *MongoDBHandler) TotalCount() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	totalCount, err := mh.collection.CountDocuments(ctx, mh.query)
	if err != nil {
		return 0, err
	}

	return totalCount, nil
}

// a query must be provided beforehand / or NOT
func (mh *MongoDBHandler) Find() (MongoResults, MongoError) {

	// connection to the mongodb server
	if err := mh.getConnection(); err.Status {
		return MongoResults{}, err
	}
	// 5 sec timeout for the context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find()
	if len(mh.sort) > 0 {
		opts.SetSort(mh.sort)
	}

	// uses the mh.query chained to the handler
	cur, err := mh.collection.Find(ctx, mh.query, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return MongoResults{}, mh.newMongoError(404, "No results found")
		}
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}
	defer cur.Close(ctx)

	// results slice
	var results []map[string]interface{}

	// decode documents
	for cur.Next(ctx) {
		var result map[string]interface{}
		if err := cur.Decode(&result); err != nil {
			return MongoResults{}, mh.newMongoError(500, err.Error())
		}
		results = append(results, result)
	}

	// iteration errors
	if err := cur.Err(); err != nil {
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}

	// Get the total count of documents
	totalCount, err := mh.TotalCount()
	if err != nil {
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}

	// should check if there's anything to be returned
	if len(results) == 0 {
		return MongoResults{}, mh.newMongoError(404, "No results found.")
	}

	// pagination handling here
	totalPages := (int(totalCount) + mh.perPage - 1) / mh.perPage
	currentPage := mh.page
	nextPage := currentPage + 1
	if nextPage > totalPages {
		nextPage = 0
	}

	plainQuery, _ := mh.Query()
	// the actual results xoxoxo
	return MongoResults{
		Status:   true,
		Code:     200,
		Database: mh.dbName,
		Table:    mh.tableName,
		Count:    totalCount,
		Results:  results,
		Pagination: MongoResultPagination{
			TotalPages:  totalPages,
			CurrentPage: currentPage,
			NextPage:    nextPage,
			LastPage:    totalPages,
			PerPage:     mh.perPage,
		},
		Query: plainQuery,
	}, MongoError{}
}

// map the erros and the code
// to avoid repetitions and code
// cluttering
func (mh *MongoDBHandler) newMongoError(code int, err string) MongoError {
	plainQuery, _ := mh.Query()
	return MongoError{
		Status:   false,
		Code:     code,
		Database: mh.dbName,
		Table:    mh.tableName,
		Error:    err,
		Query:    plainQuery,
	}
}

// takes a criteria
// and a slice of data

func (mh *MongoDBHandler) Update(update interface{}) (MongoOperationsResult, MongoError) {
	if err := mh.getConnection(); err.Status {
		return MongoOperationsResult{}, err
	}

	if mh.useTimestamps {
		switch d := update.(type) {
		case []map[string]interface{}:
			for i := range d {
				d[i] = mh.appendTimestamps(d[i]).(map[string]interface{})
			}
		case map[string]interface{}:
			update = mh.appendTimestamps(d).(map[string]interface{})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mh.collection.UpdateMany(ctx, mh.query, update)
	if err != nil {
		return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
	}

	return mh.newMongoOperations(200, true, "update", "Update performed"), MongoError{}
}

func (mh *MongoDBHandler) newMongoOperations(code int, status bool, operation string, message string) MongoOperationsResult {
	query, _ := mh.Query()

	return MongoOperationsResult{
		Status:    status,
		Code:      code,
		Database:  mh.dbName,
		Table:     mh.tableName,
		Operation: operation,
		Message:   message,
		Query:     query,
	}
}

func (mh *MongoDBHandler) UpdateByID(recordId string, data interface{}) (MongoOperationsResult, MongoError) {
	if err := mh.getConnection(); err.Status {
		return MongoOperationsResult{}, err
	}

	if mh.useTimestamps {
		switch d := data.(type) {
		case []map[string]interface{}:
			for i := range d {
				d[i] = mh.appendTimestamps(d[i]).(map[string]interface{})
			}
		case map[string]interface{}:
			data = mh.appendTimestamps(d).(map[string]interface{})
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mh.collection.UpdateByID(ctx, recordId, data)
	if err != nil {
		return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
	}

	return mh.newMongoOperations(200, true, "updateById", "Update performed for record id: "+recordId), MongoError{}
}

// basically, delete where
func (mh *MongoDBHandler) Delete(filter interface{}) (MongoOperationsResult, MongoError) {
	if err := mh.getConnection(); err.Status {
		return MongoOperationsResult{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mh.collection.DeleteMany(ctx, filter)
	if err != nil {
		return mh.newMongoOperations(500, false, "delete", "Something happened while deleting: "+err.Error()), MongoError{}
	}

	return mh.newMongoOperations(200, false, "delete", "Document deleted"), MongoError{}
}

// map the operator with the value
// to reduce code cluttering
func mapOperators(operator string, value interface{}) (interface{}, error) {
	operatorMap := map[string]interface{}{
		"=":    value,
		"!=":   map[string]interface{}{"$ne": value},
		"<":    map[string]interface{}{"$lt": value},
		"<=":   map[string]interface{}{"$lte": value},
		">":    map[string]interface{}{"$gt": value},
		">=":   map[string]interface{}{"$gte": value},
		"like": map[string]interface{}{"$regex": value},
	}

	if _, ok := operatorMap[operator]; ok {
		return operatorMap[operator], nil
	}

	return nil, errors.New("unrecognized operator")
}

// spits out the query
func (mh *MongoDBHandler) Query() (string, error) {
	if len(mh.aggregationPipeline) > 0 {
		pipeline := make([]interface{}, len(mh.aggregationPipeline))
		for i, stage := range mh.aggregationPipeline {
			pipeline[i] = bson.D{{Key: stage.StageName, Value: stage.Params}}
		}

		query := bson.D{}
		if len(mh.query) > 0 {
			query = append(query, bson.E{Key: "$match", Value: mh.query})
		}

		if len(mh.sort) > 0 {
			query = append(query, bson.E{Key: "$sort", Value: mh.sort})
		}

		// will be coming back to this
		if len(mh.aggregationPipeline) > 0 {
			var groupContents string
			for _, stage := range mh.aggregationPipeline {
				if stage.StageName == "$group" {
					fmt.Println(stage.Params)
				}
			}

			if groupContents != "" {
				query = append(query, bson.E{Key: "$group", Value: bson.D{{Key: "_id", Value: groupContents}}})
			}
		}

		jsonData, err := bson.MarshalExtJSON(query, false, false)
		if err != nil {
			return "", err
		}

		return string(jsonData), nil
	} else {
		if len(mh.query) > 0 {
			if len(mh.sort) > 0 {
				sortCriteria := make([]map[string]interface{}, 0)

				for _, element := range mh.sort {
					sortCriteria = append(sortCriteria, map[string]interface{}{element.Key: element.Value})
				}
				mh.query["$sort"] = sortCriteria
			}

			jsonData, err := json.Marshal(mh.query)

			if err != nil {
				return "", err
			}

			return string(jsonData), nil
		}

	}
	return "", errors.New("no query provided")
}

// not finished yet
func (mh *MongoDBHandler) Aggregate() (MongoResults, MongoError) {
	if err := mh.getConnection(); err.Status {
		return MongoResults{}, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Aggregate()
	var pipeline []interface{}
	for _, stage := range mh.aggregationPipeline {
		pipeline = append(pipeline, bson.D{{Key: stage.StageName, Value: stage.Params}})
	}

	cur, err := mh.collection.Aggregate(ctx, pipeline, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return MongoResults{}, mh.newMongoError(404, "No results found")
		}
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}
	defer cur.Close(ctx)

	var results []map[string]interface{}

	for cur.Next(ctx) {
		var result map[string]interface{}
		if err := cur.Decode(&result); err != nil {
			return MongoResults{}, mh.newMongoError(500, err.Error())
		}
		results = append(results, result)
	}

	if err := cur.Err(); err != nil {
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}

	totalCount, err := mh.TotalCount()
	if err != nil {
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}

	if len(results) == 0 {
		return MongoResults{}, mh.newMongoError(404, "No results found.")
	}

	totalPages := (int(totalCount) + mh.perPage - 1) / mh.perPage
	currentPage := mh.page
	nextPage := currentPage + 1
	if nextPage > totalPages {
		nextPage = 0
	}

	return MongoResults{
		Status:   true,
		Code:     200,
		Database: mh.dbName,
		Table:    mh.tableName,
		Count:    totalCount,
		Results:  results,
		Pagination: MongoResultPagination{
			TotalPages:  totalPages,
			CurrentPage: currentPage,
			NextPage:    nextPage,
			LastPage:    totalPages,
			PerPage:     mh.perPage,
		},
	}, MongoError{}
}

func (mh *MongoDBHandler) AddAggregationStage(stage AggregationStage) *MongoDBHandler {
	mh.aggregationPipeline = append(mh.aggregationPipeline, stage)
	return mh
}
func (mh *MongoDBHandler) ClearAggregationPipeline() *MongoDBHandler {
	mh.aggregationPipeline = nil
	return mh
}
