package driver

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoHandler struct {
	debug         bool
	limit         int
	perPage       int
	page          int
	sort          []primitive.E
	query         map[string]interface{}
	host          string
	port          string
	dbName        string
	tableName     string
	username      string
	password      string
	useTimestamps bool
	timeNow       time.Time
	client        *mongo.Client
	db            *mongo.Database
	collection    *mongo.Collection
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
}

// results error
type MongoResultsError struct {
	Status   bool
	Code     int
	Database string
	Table    string
	Error    string
}

// initializes the mongo handler driver
// similar to how constructors work
func NewMongoHandler() *MongoHandler {
	host := helpers.GetEnv("MONGO_DB_HOST", "localhost")
	port := helpers.GetEnv("MONGO_DB_PORT", "27017")
	dbName := helpers.GetEnv("MONGO_DB_NAME=", "test")
	tableName := helpers.GetEnv("MONGO_DB_TABLE", "test")
	username := helpers.GetEnv("MONGO_DB_USERNAME", "admin")
	password := helpers.GetEnv("MONGO_DB_PASSWORD", "admin")
	useTimestamps, _ := strconv.ParseBool(helpers.GetEnv("HANDLER_USE_TIMESTAMPS", "true"))
	debug, _ := strconv.ParseBool(helpers.GetEnv("HANDLER_DEBUG", "false"))

	return &MongoHandler{
		debug:         debug,
		limit:         0,
		perPage:       10,
		page:          1,
		sort:          []primitive.E{},
		query:         make(map[string]interface{}),
		host:          host,
		port:          port,
		dbName:        dbName,
		tableName:     tableName,
		username:      username,
		password:      password,
		useTimestamps: useTimestamps,
		timeNow:       time.Now(),
		client:        nil,
		db:            nil,
		collection:    nil,
	}
}

// handles connectivity
func (mh *MongoHandler) getConnection() error {
	if mh.client != nil {
		return nil
	}

	clientOptions := options.Client().ApplyURI("mongodb://" + mh.host + ":" + mh.port).
		SetAuth(options.Credential{
			Username: mh.username,
			Password: mh.password,
		}).
		SetSocketTimeout(2 * time.Second)

	client, err := mongo.Connect(context.Background(), clientOptions)
	if err != nil {
		return err
	}

	err = client.Ping(context.Background(), nil)
	if err != nil {
		return err
	}

	mh.client = client
	mh.db = mh.client.Database(mh.dbName)
	mh.collection = mh.db.Collection(mh.tableName)

	return nil
}

// set the table / collection name
func (mh *MongoHandler) FromTable(tableName string) *MongoHandler {
	mh.tableName = tableName
	return mh
}

// set table / collection name
func (mh *MongoHandler) IntoTable(tableName string) *MongoHandler {
	mh.tableName = tableName
	return mh
}

// set db name
func (mh *MongoHandler) FromDB(dbName string) *MongoHandler {
	mh.dbName = dbName
	return mh
}

// set  db name
func (mh *MongoHandler) IntoDB(dbName string) *MongoHandler {
	mh.dbName = dbName
	return mh
}

func (mh *MongoHandler) PerPage(perPage int) *MongoHandler {
	mh.perPage = perPage
	return mh
}

func (mh *MongoHandler) Page(page int) *MongoHandler {
	mh.page = page
	return mh
}

// Where contraint
func (mh *MongoHandler) Where(field string, operator string, value interface{}) *MongoHandler {
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

	mh.query[field] = mappedValue

	return mh
}

// and where constraint
func (mh *MongoHandler) AndWhere(field, operator string, value interface{}) *MongoHandler {
	if field == "_id" {
		objectID, err := primitive.ObjectIDFromHex(value.(string))
		if err != nil {

			return mh
		}
		value = objectID
	}

	and_condition := make(map[string]interface{})

	mappedValue, err := mapOperators(operator, value)
	if err != nil {
		fmt.Println("Error:", err)
		return mh
	}

	and_condition[field] = mappedValue

	// check for $and contraints in the query
	if _, ok := mh.query["$and"]; !ok {
		mh.query["$and"] = []interface{}{}
	}

	mh.query["$and"] = append(mh.query["$and"].([]interface{}), and_condition)

	return mh
}

// or where
func (mh *MongoHandler) OrWhere(field, operator string, value interface{}) *MongoHandler {
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

	// check for $or contraints in the query
	if _, ok := mh.query["$or"]; !ok {
		mh.query["$or"] = []interface{}{}
	}

	mh.query["$or"] = append(mh.query["$or"].([]interface{}), or_condition)

	return mh
}

// ascending / descending sorting -> 1, -1
func (mh *MongoHandler) Sort(field, order string) *MongoHandler {
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

// setting up results limit
func (mh *MongoHandler) Limit(limit int) *MongoHandler {
	mh.limit = limit
	return mh
}

// just adds created_at and updated_at timestamps
// upon insertion and or update
func (mh *MongoHandler) appendTimestamps(data interface{}) interface{} {
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

func (mh *MongoHandler) Insert(data interface{}) error {
	if err := mh.getConnection(); err != nil {
		return err
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
			return err
		}
	case map[string]interface{}:
		_, err := mh.collection.InsertOne(ctx, d)
		if err != nil {
			return err
		}
	default:
		return errors.New("unsupported data type")
	}

	return nil
}

// deletes a mongo db
func (mh *MongoHandler) DropDatabase(dbName string) error {
	if err := mh.getConnection(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mh.client.Database(dbName).Drop(ctx)
	if err != nil {
		return err
	}

	return nil
}

// deletes a mongo collection / table
func (mh *MongoHandler) DropTable(tableName string) error {
	if err := mh.getConnection(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mh.collection.Drop(ctx)
	if err != nil {
		return err
	}

	return nil
}

func (mh *MongoHandler) TotalCount() (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	totalCount, err := mh.collection.CountDocuments(ctx, mh.query)
	if err != nil {
		// Handle error
		return 0, err
	}

	return totalCount, nil
}

// a query must be provided beforehand
func (mh *MongoHandler) Find() (MongoResults, MongoResultsError) {
	// connection to the mongodb server
	if err := mh.getConnection(); err != nil {
		return MongoResults{}, mh.newMongoResultsError(500, err.Error())
	}

	// 5 sec timeout for the context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	opts := options.Find()
	if len(mh.sort) > 0 {
		opts.SetSort(mh.sort)
	}

	fmt.Println(mh.query)

	// uses the mh.query chained to the handler
	cur, err := mh.collection.Find(ctx, mh.query, opts)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return MongoResults{}, mh.newMongoResultsError(404, "No results found")
		}
		return MongoResults{}, mh.newMongoResultsError(500, err.Error())
	}
	defer cur.Close(ctx)

	// results slice
	var results []map[string]interface{}

	// decode documents
	for cur.Next(ctx) {
		var result map[string]interface{}
		if err := cur.Decode(&result); err != nil {
			return MongoResults{}, mh.newMongoResultsError(500, err.Error())
		}
		results = append(results, result)
	}

	// iteration errors
	if err := cur.Err(); err != nil {
		return MongoResults{}, mh.newMongoResultsError(500, err.Error())
	}

	// Get the total count of documents
	totalCount, err := mh.TotalCount()
	if err != nil {
		return MongoResults{}, mh.newMongoResultsError(500, err.Error())
	}

	// should check if there's anything to be returned
	if len(results) == 0 {
		return MongoResults{}, mh.newMongoResultsError(404, "No results found.")
	}

	// pagination handling here
	totalPages := (int(totalCount) + mh.perPage - 1) / mh.perPage
	currentPage := mh.page
	nextPage := currentPage + 1
	if nextPage > totalPages {
		nextPage = 0
	}

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
	}, MongoResultsError{}
}

// map the erros and the code
// to avoid repetitions and code
// cluttering
func (mh *MongoHandler) newMongoResultsError(code int, err string) MongoResultsError {
	return MongoResultsError{
		Status:   false,
		Code:     code,
		Database: mh.dbName,
		Table:    mh.tableName,
		Error:    err,
	}
}

// takes a criteria
// and a slice of data

func (mh *MongoHandler) Update(filter, update interface{}) error {
	if err := mh.getConnection(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mh.collection.UpdateMany(ctx, filter, update)
	if err != nil {
		return err
	}

	return nil
}

// basically, delete where
func (mh *MongoHandler) Delete(filter interface{}) error {
	if err := mh.getConnection(); err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, err := mh.collection.DeleteMany(ctx, filter)
	if err != nil {
		return err
	}

	return nil
}

// map the operator with the value
// to reduce code cluttering
func mapOperators(operator string, value interface{}) (interface{}, error) {
	operatorMap := map[string]interface{}{
		"=":     value,
		"!=":    map[string]interface{}{"$ne": value},
		"<":     map[string]interface{}{"$lt": value},
		"<=":    map[string]interface{}{"$lte": value},
		">":     map[string]interface{}{"$gt": value},
		">=":    map[string]interface{}{"$gte": value},
		"_like": map[string]interface{}{"$regex": value, "options": "i"},
	}

	if _, ok := operatorMap[operator]; ok {
		return operatorMap[operator], nil
	}

	return nil, errors.New("unrecognized operator")
}
