package driver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

var canConnect bool

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
	logger              *log.Logger
}

// used for results mapping
type MongoResultPagination struct {
	TotalPages  int
	CurrentPage int
	NextPage    int
	PrevPage    int
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

type MongoDatabaseListResult struct {
	Status    bool
	Code      int
	Databases []string
}

type MongoTablesListResult struct {
	Status   bool
	Code     int
	Database string
	Tables   []string
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

type SingleMongoResult struct {
	Status   bool
	Code     int
	Database string
	Table    string
	IdType   string
	Result   interface{}
}

// initializes the mongo handler driver
// similar to how constructors work
func MongoDB() *MongoDBHandler {
	logger := log.New(os.Stdout, "[MONGO-DB-DRIVER]: ", log.Ldate|log.Ltime)

	canConnect = true
	host := helpers.GetEnv("MONGO_DB_HOST", "localhost")
	port := helpers.GetEnv("MONGO_DB_PORT", "27017")
	dbName := helpers.GetEnv("MONGO_DB_NAME", "test")
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
		logger:         logger,
	}
}

func convertMongoID(id interface{}) string {
	if objID, ok := id.(primitive.ObjectID); ok {
		return objID.Hex()
	}
	return fmt.Sprintf("%v", id)
}

func (mh *MongoDBHandler) TestMongoConnection() bool {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return false
		}
	}
	return true
}

// handles connectivity
func (mh *MongoDBHandler) getConnection() MongoError {
	mh.logger.Println("Connecting to Mongo Server...")

	if mh.client != nil {
		mh.logger.Println("Connection stil active, using previous connection...")
		// use the previously initialized connection
		return MongoError{}
	}

	clientOptions := options.Client().ApplyURI("mongodb://" + mh.host + ":" + mh.port).
		SetAuth(options.Credential{
			Username: mh.username,
			Password: mh.password,
		}).
		SetMaxPoolSize(10).
		SetSocketTimeout(2 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		mh.logger.Printf("Connection to MongoDB server failed: " + err.Error())
		return mh.newMongoError(500, err.Error())
	}

	ctxPing, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelPing()

	err = client.Ping(ctxPing, nil)
	if err != nil {
		mh.logger.Printf("Ping to MongoDB server failed: " + err.Error())
		return mh.newMongoError(500, err.Error())
	}

	mh.client = client
	mh.db = mh.client.Database(mh.dbName)
	mh.collection = mh.db.Collection(mh.tableName)

	mh.logger.Println("Connection to Mongo Server succesful!")

	return MongoError{}
}

// set db name
func (mh *MongoDBHandler) DB(dbName string) *MongoDBHandler {
	mh.dbName = dbName
	if mh.client != nil {
		mh.db = mh.client.Database(dbName)
	}

	return mh
}

// set table / collection name
func (mh *MongoDBHandler) Table(tableName string) *MongoDBHandler {
	mh.tableName = tableName
	if mh.client != nil {
		mh.collection = mh.db.Collection(tableName)
	}
	return mh
}

// provide page if possible
func (mh *MongoDBHandler) Page(page int) *MongoDBHandler {
	mh.page = page
	return mh
}

// provide the limit of records per page
func (mh *MongoDBHandler) PerPage(perPage int) *MongoDBHandler {
	if perPage <= 300 {
		mh.perPage = perPage
	}
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

	fmt.Println(mh.dbName)
	fmt.Println(mh.tableName)

	if err := mh.getConnection(); err.Error != "" {
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
	case []interface{}:
		var interfaceSlice []interface{}
		for _, item := range d {
			if itemMap, ok := item.(map[string]interface{}); ok {
				interfaceSlice = append(interfaceSlice, itemMap)
			} else {
				return MongoOperationsResult{}, mh.newMongoError(400, "unsupported data type in array")
			}
		}
		_, err := mh.collection.InsertMany(ctx, interfaceSlice)
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
	if mh.client == nil {
		// connection to the mongodb server
		if err := mh.getConnection(); err.Error != "" {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	dbs, err := mh.client.ListDatabaseNames(ctx, bson.M{})
	if err != nil {
		return mh.newMongoError(500, "Unable to list databases. Error: "+err.Error())
	}

	// Check if dbName is in the list of databases
	dbExists := false
	for _, db := range dbs {
		if db == dbName {
			dbExists = true
			break
		}
	}

	if !dbExists {
		return mh.newMongoError(404, "Database not found: "+dbName)
	}

	err = mh.client.Database(dbName).Drop(ctx)
	if err != nil {
		return mh.newMongoError(500, "Unable to drop database: "+dbName+". Error: "+err.Error())
	}

	return MongoError{}
}

// deletes a mongo collection / table
func (mh *MongoDBHandler) DropTable(dbName string, collectionName string) MongoError {
	if mh.client == nil {
		// connection to the mongodb server
		if err := mh.getConnection(); err.Error != "" {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	databases, databasesErr := mh.ListDatabases()
	if databasesErr.Error != "" {
		return mh.newMongoError(500, "Unable to retrieve existing database list: "+databasesErr.Error)
	}

	dbExists := false
	for _, db := range databases.Databases {
		if db == dbName {
			dbExists = true
			break
		}
	}

	if !dbExists {
		return mh.newMongoError(404, "Database not found: "+dbName)
	}

	mh.db = mh.client.Database(dbName)

	tables, tablesErr := mh.ListCollections(dbName)

	if tablesErr.Error != "" {
		return mh.newMongoError(500, "Unable to retrieve existing table list in specified database: "+tablesErr.Error)
	}

	tableExists := false

	for _, table := range tables.Tables {
		if table == collectionName {
			tableExists = true
			break
		}
	}

	if !tableExists {
		return mh.newMongoError(404, "Table / collection: "+collectionName+" not found in database: "+dbName)
	}

	mh.collection = mh.db.Collection(collectionName)

	err := mh.collection.Drop(ctx)
	if err != nil {
		return mh.newMongoError(500, "Unable to drop collection: "+collectionName+"Error: "+err.Error())
	}

	return MongoError{}
}

func (mh *MongoDBHandler) TotalCount() (int64, error) {
	if mh.collection == nil {
		return 0, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// for case insensitive searching
	countOptions := options.Count().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // Case-insensitive
	})
	totalCount, err := mh.collection.CountDocuments(ctx, mh.query, countOptions)
	if err != nil {
		return 0, err
	}

	return totalCount, nil
}
func (mh *MongoDBHandler) AndAll(queryInput [][]interface{}) *MongoDBHandler {
	if len(queryInput) > 0 {
		for _, item := range queryInput {
			mh.Where(item[0].(string), item[1].(string), item[2])
		}
	}

	return mh
}
func (mh *MongoDBHandler) OrAll(queryInput [][]interface{}) *MongoDBHandler {
	if len(queryInput) > 0 {
		for _, item := range queryInput {
			mh.OrWhere(item[0].(string), item[1].(string), item[2])
		}
	}

	return mh
}

func (mh *MongoDBHandler) SortAll(sortInput [][]interface{}) *MongoDBHandler {
	if len(sortInput) > 0 {
		for _, item := range sortInput {
			mh.SortBy(item[0].(string), item[1].(string))
		}
	}

	return mh
}

// a query must be provided beforehand / or NOT
func (mh *MongoDBHandler) Find() (MongoResults, MongoError) {
	if mh.client == nil {
		// connection to the mongodb server
		if err := mh.getConnection(); err.Error != "" {
			return MongoResults{}, err
		}
	}

	// 5 sec timeout for the context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// need case insensitive so we're applyhing this
	opts := options.Find().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // Case-insensitive
	})

	if len(mh.sort) > 0 {
		opts.SetSort(mh.sort)
	}

	opts.SetLimit(int64(mh.perPage))
	opts.SetSkip(int64((mh.page - 1) * mh.perPage))

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
	prevPage := 1
	nextPage := 1

	if currentPage > 1 {
		prevPage = currentPage - 1
	}
	if currentPage < totalPages {
		nextPage = currentPage + 1
	}

	// GRAB THE QUERY
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
			PrevPage:    prevPage,
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
	if mh.client == nil {
		// connection to the mongodb server
		if err := mh.getConnection(); err.Error != "" {
			return MongoOperationsResult{}, err
		}
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

	opts := options.Update().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // Case-insensitive
	})

	_, err := mh.collection.UpdateMany(ctx, mh.query, bson.M{"$set": update}, opts)
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
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoOperationsResult{}, err
		}
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

	var filter bson.M

	findRecord, findRecordErr := mh.FindById(recordId)

	if findRecordErr.Error != "" {
		return MongoOperationsResult{}, findRecordErr
	}

	update := bson.M{"$set": data}
	if findRecord.IdType == "mongo" {
		objID, err := primitive.ObjectIDFromHex(recordId)
		if err != nil {
			return MongoOperationsResult{}, mh.newMongoError(500, "Unable to convert string to Mongo ID: "+err.Error())
		}
		filter = bson.M{"_id": objID}

	} else {
		filter = bson.M{"_id": recordId}
	}

	_, err := mh.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
	}

	return mh.newMongoOperations(200, true, "updateById", "Update performed"), MongoError{}
}

func (mh *MongoDBHandler) FindById(recordId string) (SingleMongoResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return SingleMongoResult{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var filter bson.M
	var result bson.M
	var err error

	// try using regular mongo id
	objID, objErr := primitive.ObjectIDFromHex(recordId)
	if objErr == nil {
		filter = bson.M{"_id": objID}
		err = mh.collection.FindOne(ctx, filter).Decode(&result)
		if err == nil {
			resultMap := make(map[string]interface{})
			for k, v := range result {
				if k == "_id" {
					resultMap["id"] = convertMongoID(v)
				} else {
					resultMap[k] = v
				}
			}
			return SingleMongoResult{
				Status:   true,
				Code:     200,
				IdType:   "mongo",
				Database: mh.dbName,
				Table:    mh.tableName,
				Result:   resultMap,
			}, MongoError{}
		}
	}

	// try using regular string as an ID
	filter = bson.M{"_id": recordId}
	err = mh.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return SingleMongoResult{}, mh.newMongoError(404, "Record not found!")
	}

	resultMap := make(map[string]interface{})
	for k, v := range result {
		if k == "_id" {
			resultMap["id"] = convertMongoID(v)
		} else {
			resultMap[k] = v
		}
	}
	return SingleMongoResult{
		Status:   true,
		Code:     200,
		IdType:   "string",
		Database: mh.dbName,
		Table:    mh.tableName,
		Result:   resultMap,
	}, MongoError{}

}

// wipes a record by mongo_id
func (mh *MongoDBHandler) DeleteById(recordId string) (MongoOperationsResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoOperationsResult{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var filter bson.M
	findRecord, findRecordErr := mh.FindById(recordId)

	if findRecordErr.Error != "" {
		return MongoOperationsResult{}, findRecordErr
	}

	if findRecord.IdType == "mongo" {
		objID, err := primitive.ObjectIDFromHex(recordId)
		if err != nil {
			return MongoOperationsResult{}, mh.newMongoError(500, "Unable to convert string to Mongo ID: "+err.Error())
		}
		filter = bson.M{"_id": objID}

	} else {
		filter = bson.M{"_id": recordId}
	}

	_, err := mh.collection.DeleteOne(ctx, filter)
	if err != nil {
		return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
	}

	return mh.newMongoOperations(200, true, "deleteByid", "Delete operation performed."), MongoError{}
}

// basically, delete where
func (mh *MongoDBHandler) Delete() (MongoOperationsResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoOperationsResult{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Delete().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2, // Case-insensitive
	})

	_, err := mh.collection.DeleteMany(ctx, mh.query, opts)
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

func (mh *MongoDBHandler) First() {

}

// not finished yet
func (mh *MongoDBHandler) Aggregate() (MongoResults, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoResults{}, err
		}
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

func (mh *MongoDBHandler) ListDatabases() (MongoDatabaseListResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoDatabaseListResult{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	databases, err := mh.client.ListDatabaseNames(ctx, bson.D{}) // looks weird, but it requires an empty document bson.D{}
	if err != nil {
		return MongoDatabaseListResult{}, mh.newMongoError(500, err.Error())
	}

	if len(databases) < 1 {
		return MongoDatabaseListResult{}, mh.newMongoError(404, "No databases found for this server.")
	}

	return MongoDatabaseListResult{
		Status:    true,
		Code:      200,
		Databases: databases,
	}, MongoError{}
}

func (mh *MongoDBHandler) ListCollections(dbName string) (MongoTablesListResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoTablesListResult{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// List collection names
	collections, err := mh.client.Database(dbName).ListCollectionNames(ctx, bson.M{})
	if err != nil {
		return MongoTablesListResult{}, mh.newMongoError(500, "Unable to list collections in database: "+dbName+". Error: "+err.Error())
	}

	if len(collections) < 1 {
		return MongoTablesListResult{}, mh.newMongoError(200, "No collections found in the database.")
	}

	return MongoTablesListResult{
		Status:   true,
		Code:     200,
		Database: dbName,
		Tables:   collections,
	}, MongoError{}
}

func (mh *MongoDBHandler) ResetQuery() *MongoDBHandler {
	if len(mh.query) > 0 {
		mh.query = make(map[string]interface{})
	}
	return mh
}

func (mh *MongoDBHandler) ResetSort() *MongoDBHandler {
	if len(mh.sort) > 0 {
		mh.sort = []primitive.E{}
	}
	return mh
}

func (mh *MongoDBHandler) ResetState() *MongoDBHandler {
	if len(mh.query) > 0 {
		mh.query = make(map[string]interface{})
	}

	if len(mh.sort) > 0 {
		mh.sort = []primitive.E{}
	}

	if mh.dbName != "" {
		mh.dbName = ""
	}

	if mh.tableName != "" {
		mh.tableName = ""
	}

	return mh
}

// func (mh *MongoDBHandler) ExecuteQuery(queryString string) {
// 	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
// 	defer cancel()

// 	// should be able to take the query in string format
// 	// convert it to BSON so it canbe executed
// 	// in mongo
// 	// then, should be able to return whatever the result was
// }
