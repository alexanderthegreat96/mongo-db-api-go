package driver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/alexanderthegreat96/envparser/v2"
	"github.com/emirpasic/gods/maps/linkedhashmap"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func MongoDB() *MongoDBHandler {
	logger := log.New(os.Stdout, "[MONGO-DB-DRIVER]: ", log.Ldate|log.Ltime)
	env := envparser.NewEnvParser(
		envparser.WithFilename(".env"),
		envparser.WithRootPath(true),
	)
	if err := env.GetError(); err != "" {
		logger.Fatalf("Failed loading .env: %s", err)
	}

	rawHost, _ := env.GetValue("MONGO_DB_HOST", "string", "localhost")
	rawPort, _ := env.GetValue("MONGO_DB_PORT", "string", "27017")
	rawDBName, _ := env.GetValue("MONGO_DB_NAME", "string", "test")
	rawTableName, _ := env.GetValue("MONGO_DB_TABLE", "string", "test")
	rawUsername, _ := env.GetValue("MONGO_DB_USERNAME", "string", "admin")
	rawPassword, _ := env.GetValue("MONGO_DB_PASSWORD", "string", "admin")
	rawUseTimestamps, _ := env.GetValue("HANDLER_USE_TIMESTAMPS", "bool", true)
	rawDebug, _ := env.GetValue("HANDLER_DEBUG", "bool", false)

	host := rawHost.(string)
	port := rawPort.(string)
	dbName := rawDBName.(string)
	tableName := rawTableName.(string)
	username := rawUsername.(string)
	password := rawPassword.(string)
	useTimestamps := rawUseTimestamps.(bool)
	debug := rawDebug.(bool)

	return &MongoDBHandler{
		debug:          debug,
		limit:          0,
		perPage:        10,
		page:           1,
		sort:           []primitive.E{},
		query:          make(map[string]any),
		aggregateQuery: []any{},
		multipleWheres: false,

		host:          host,
		port:          port,
		dbName:        dbName,
		tableName:     tableName,
		username:      username,
		password:      password,
		useTimestamps: useTimestamps,

		timeNow:    time.Now(),
		client:     nil,
		db:         nil,
		collection: nil,
		logger:     logger,
	}
}

// convertMongoID converts an ObjectID (or other type) to a string.
func convertMongoID(id any) string {
	if objID, ok := id.(primitive.ObjectID); ok {
		return objID.Hex()
	}
	return fmt.Sprintf("%v", id)
}

// CanConnectToMongo tests the connection.
func (mh *MongoDBHandler) CanConnectToMongo() bool {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return false
		}
	}
	return true
}

// getConnection sets up the connection if not already done.
func (mh *MongoDBHandler) getConnection() MongoError {
	mh.logger.Println("Connecting to Mongo Server...")

	if mh.client != nil {
		mh.logger.Println("Connection still active, using previous connection...")
		return MongoError{}
	}

	clientOptions := options.Client().ApplyURI("mongodb://" + mh.host + ":" + mh.port).
		SetAuth(options.Credential{
			Username: mh.username,
			Password: mh.password,
		}).
		SetMaxPoolSize(15).
		SetSocketTimeout(3 * time.Second)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		mh.logger.Printf("Connection to MongoDB server failed: %s", err.Error())
		return mh.newMongoError(500, err.Error())
	}

	ctxPing, cancelPing := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancelPing()

	if err = client.Ping(ctxPing, nil); err != nil {
		mh.logger.Printf("Ping to MongoDB server failed: %s", err.Error())
		return mh.newMongoError(500, err.Error())
	}

	mh.client = client
	mh.db = mh.client.Database(mh.dbName)
	mh.collection = mh.db.Collection(mh.tableName)

	mh.logger.Println("Connection to Mongo Server successful!")
	return MongoError{}
}

// DB sets the database name.
func (mh *MongoDBHandler) DB(dbName string) *MongoDBHandler {
	mh.dbName = dbName
	if mh.client != nil {
		mh.db = mh.client.Database(dbName)
	}
	return mh
}

// Table sets the collection (table) name.
func (mh *MongoDBHandler) Table(tableName string) *MongoDBHandler {
	mh.tableName = tableName
	if mh.client != nil && mh.db != nil {
		mh.collection = mh.db.Collection(tableName)
	}
	return mh
}

func (mh *MongoDBHandler) Page(page int) *MongoDBHandler {
	mh.page = page
	return mh
}

func (mh *MongoDBHandler) PerPage(perPage int) *MongoDBHandler {
	if perPage > 0 && perPage <= 300 {
		mh.perPage = perPage
	}
	return mh
}

func (mh *MongoDBHandler) InnerPage(p int) *MongoDBHandler {
	if p < 1 {
		p = 1
	}
	mh.innerPage = p
	return mh
}
func (mh *MongoDBHandler) InnerPerPage(n int) *MongoDBHandler {
	if n < 1 || n > 300 {
		n = 10
	}
	mh.innerPerPage = n
	return mh
}

// Where adds a filter constraint.
// If the field is "_id", it ensures that the value is a valid ObjectID.
func (mh *MongoDBHandler) Where(field string, operator string, value any) *MongoDBHandler {
	if field == "_id" {
		strVal, ok := value.(string)
		if !ok {
			mh.logger.Println("Error: _id value must be a string")
			return mh
		}
		objectID, err := primitive.ObjectIDFromHex(strVal)
		if err != nil {
			mh.logger.Printf("Error converting _id value: %s", err.Error())
			return mh
		}
		value = objectID
	}

	mappedValue, err := MapOperators(operator, value)
	if err != nil {
		mh.logger.Printf("Error in operator mapping: %s", err.Error())
		return mh
	}

	if mh.query == nil {
		mh.query = make(map[string]any)
	}

	if !mh.multipleWheres && len(mh.query) == 0 {
		mh.query[field] = mappedValue
		return mh
	}

	if !mh.multipleWheres && len(mh.query) > 0 {
		var andConditions []any
		for k, v := range mh.query {
			andConditions = append(andConditions, map[string]any{k: v})
		}
		andConditions = append(andConditions, map[string]any{field: mappedValue})
		mh.query = map[string]any{"$and": andConditions}
		mh.multipleWheres = true
		return mh
	}

	// Case 3: Already using $and
	if mh.multipleWheres {
		existingAnd, ok := mh.query["$and"].([]any)
		if !ok || existingAnd == nil {
			existingAnd = []any{}
		}
		existingAnd = append(existingAnd, map[string]any{field: mappedValue})
		mh.query["$and"] = existingAnd
		return mh
	}

	return mh
}

// OrWhere adds an OR filter constraint.
func (mh *MongoDBHandler) OrWhere(field, operator string, value any) *MongoDBHandler {
	if field == "_id" {
		strVal, ok := value.(string)
		if !ok {
			mh.logger.Println("Error: _id value must be a string")
			return mh
		}
		objectID, err := primitive.ObjectIDFromHex(strVal)
		if err != nil {
			mh.logger.Printf("Error converting _id value: %s", err.Error())
			return mh
		}
		value = objectID
	}
	orCondition := make(map[string]any)
	mappedValue, err := MapOperators(operator, value)
	if err != nil {
		mh.logger.Printf("Error in operator mapping: %s", err.Error())
		return mh
	}
	orCondition[field] = mappedValue

	if _, ok := mh.query["$or"]; !ok {
		mh.query["$or"] = []any{}
	}
	mh.query["$or"] = append(mh.query["$or"].([]any), orCondition)
	return mh
}

// SortBy adds a sort order for the given field.
func (mh *MongoDBHandler) SortBy(field, order string) *MongoDBHandler {
	var sortOrder int32 = -1
	if strings.ToLower(order) == "asc" {
		sortOrder = 1
	}
	mh.sort = append(mh.sort, primitive.E{Key: field, Value: sortOrder})
	return mh
}

// GroupBy builds an aggregation pipeline that groups documents by the given field,
// then applies optional sorting and pagination on the grouped results.
func (mh *MongoDBHandler) GroupBy(field string) *MongoDBHandler {
	// 1) clamp outer page & inner page parameters
	if mh.page < 1 {
		mh.page = 1
	}

	if mh.perPage < 1 {
		mh.perPage = 10
	}

	if mh.innerPage < 1 {
		mh.innerPage = 1
	}

	if mh.innerPerPage < 1 {
		mh.innerPerPage = 10
	}

	// 2) calculate skips & limits
	groupSkip := (mh.page - 1) * mh.perPage
	groupLimit := mh.perPage
	recordSkip := (mh.innerPage - 1) * mh.innerPerPage
	recordLimit := mh.innerPerPage

	// 3) build the pipeline
	var pipeline []any

	// a) match outer filters
	if len(mh.query) > 0 {
		pipeline = append(pipeline, bson.M{"$match": mh.query})
	}
	// b) sort before grouping
	if len(mh.sort) > 0 {
		pipeline = append(pipeline, bson.M{"$sort": mh.sort})
	}
	// c) group into arrays + count
	pipeline = append(pipeline, bson.M{"$group": bson.M{
		"_id":           "$" + field,
		"records":       bson.M{"$push": "$$ROOT"},
		"total_records": bson.M{"$sum": 1},
	}})
	// d) page the groups
	pipeline = append(pipeline, bson.M{"$skip": groupSkip})
	pipeline = append(pipeline, bson.M{"$limit": groupLimit})
	// e) slice each group's records + inner pagination metadata
	totalPagesExpr := bson.M{"$ceil": bson.M{
		"$divide": []any{"$total_records", recordLimit},
	}}

	pipeline = append(pipeline, bson.M{"$project": bson.M{
		"_id":           1,
		"total_records": 1,
		"records":       bson.M{"$slice": []any{"$records", recordSkip, recordLimit}},
		"inner_pagination": bson.M{
			// these two were previously dropped, now forced into output:
			"current_page": bson.M{"$literal": mh.innerPage},
			"per_page":     bson.M{"$literal": recordLimit},
			// these were already working as expressions:
			"total_pages": totalPagesExpr,
			"last_page":   totalPagesExpr,
			"next_page": bson.M{"$cond": []any{
				bson.M{"$lt": []any{mh.innerPage, totalPagesExpr}},
				mh.innerPage + 1,
				mh.innerPage,
			}},
			"prev_page": bson.M{"$cond": []any{
				bson.M{"$gt": []any{mh.innerPage, 1}},
				mh.innerPage - 1,
				1,
			}},
		},
	}})

	// applies sorting again
	if len(mh.sort) > 0 {
		pipeline = append(pipeline, bson.M{"$sort": mh.sort})
	}

	mh.aggregateQuery = pipeline
	return mh
}

func (mh *MongoDBHandler) Limit(limit int) *MongoDBHandler {
	mh.limit = limit
	return mh
}

func toJsonBytes(data any) (string, error) {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(jsonData), nil
}

// appendTimestamps adds created_at and updated_at timestamps.
func (mh *MongoDBHandler) appendTimestamps(data any, operation string) any {
	if !mh.useTimestamps {
		return data
	}
	addOrUpdateTimestamps := func(itemMap map[string]any) {
		if operation == "insert" {
			if _, exists := itemMap["created_at"]; !exists {
				itemMap["created_at"] = mh.timeNow
			}
		}
		if operation == "insert" || operation == "update" {
			itemMap["updated_at"] = mh.timeNow
		}
	}
	switch d := data.(type) {
	case []any:
		for i, item := range d {
			if itemMap, ok := item.(map[string]any); ok {
				addOrUpdateTimestamps(itemMap)
				d[i] = itemMap
			}
		}
	case []map[string]any:
		for i, itemMap := range d {
			addOrUpdateTimestamps(itemMap)
			d[i] = itemMap
		}
	case map[string]any:
		addOrUpdateTimestamps(d)
	}
	return data
}

// appendTimestampForCreatedAt uses a linkedhashmap to add timestamps.
func (mh *MongoDBHandler) appendTimestampForCreatedAt(data map[string]any) map[string]any {
	if mh.useTimestamps {
		jsonBytes, err := toJsonBytes(data)
		if err != nil {
			return data
		}
		hm := linkedhashmap.New()
		_ = hm.FromJSON([]byte(jsonBytes))
		hm.Put("created_at", mh.timeNow)
		hm.Put("updated_at", mh.timeNow)
		reEncodedBytes, _ := hm.ToJSON()
		if result, err := ConvertJsonToMap(string(reEncodedBytes)); err == nil {
			return result
		}
	}
	return data
}

// chunkSlice splits a slice of map[string]any into smaller slices.
func chunkSlice(slice []map[string]any, chunkSize int) [][]map[string]any {
	var chunks [][]map[string]any
	for chunkSize < len(slice) {
		slice, chunks = slice[chunkSize:], append(chunks, slice[0:chunkSize:chunkSize])
	}
	chunks = append(chunks, slice)
	return chunks
}

// chunkInterfaceSlice splits a slice of interfaces into chunks.
func chunkInterfaceSlice(slice []any, chunkSize int) [][]any {
	var chunks [][]any
	for i := 0; i < len(slice); i += chunkSize {
		end := i + chunkSize
		if end > len(slice) {
			end = len(slice)
		}
		chunks = append(chunks, slice[i:end])
	}
	return chunks
}

// insertChunk is used internally to insert a chunk of documents concurrently.
func (mh *MongoDBHandler) insertChunk(ctx context.Context, chunk []map[string]any, wg *sync.WaitGroup, resultCh chan<- MongoOperationsResult, errCh chan<- MongoError) {
	defer wg.Done()
	var interfaceSlice []any
	for _, item := range chunk {
		if mh.useTimestamps {
			if updated, ok := mh.appendTimestamps(item, "insert").(map[string]any); ok {
				item = updated
			}
		}
		interfaceSlice = append(interfaceSlice, item)
	}
	_, err := mh.collection.InsertMany(ctx, interfaceSlice)
	if err != nil {
		errCh <- mh.newMongoError(500, err.Error())
		return
	}
	resultCh <- mh.newMongoOperations(200, true, "insert", "Chunk insert performed.")
}

// calculateBatchSize returns a batch size based on a percentage of total records.
func calculateBatchSize(totalRecords int, percentage float64) int {
	batchSize := int(float64(totalRecords) * percentage / 100.0)
	fmt.Println(batchSize)
	return batchSize
}

// countRecords returns the number of records in data.
func countRecords(data any) int {
	switch d := data.(type) {
	case []map[string]any:
		return len(d)
	case map[string]any:
		return 1
	case []any:
		return len(d)
	default:
		return 0
	}
}

// Insert performs an insert operation (supports map, slice of map, or slice of any).
func (mh *MongoDBHandler) Insert(data any) (MongoOperationsResult, MongoError) {
	if err := mh.getConnection(); err.Error != "" {
		return MongoOperationsResult{}, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	switch d := data.(type) {
	case []map[string]any:
		chunks := chunkSlice(d, 300)
		var wg sync.WaitGroup
		resultCh := make(chan MongoOperationsResult, len(chunks))
		errCh := make(chan MongoError, len(chunks))
		for _, chunk := range chunks {
			wg.Add(1)
			go mh.insertChunk(ctx, chunk, &wg, resultCh, errCh)
		}
		wg.Wait()
		close(resultCh)
		close(errCh)
		for err := range errCh {
			if err.Error != "" {
				return MongoOperationsResult{}, err
			}
		}
		return mh.newMongoOperations(200, true, "insert", "All inserts performed."), MongoError{}
	case map[string]any:
		if mh.useTimestamps {
			if updated, ok := mh.appendTimestamps(d, "insert").(map[string]any); ok {
				d = updated
			}
		}
		_, err := mh.collection.InsertOne(ctx, d)
		if err != nil {
			return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
		}
	case []any:
		chunks := chunkInterfaceSlice(d, 300)
		var wg sync.WaitGroup
		resultCh := make(chan MongoOperationsResult, len(chunks))
		errCh := make(chan MongoError, len(chunks))
		for _, chunk := range chunks {
			wg.Add(1)
			go func(chunk []any) {
				defer wg.Done()
				var interfaceSlice []any
				for _, item := range chunk {
					itemMap, ok := item.(map[string]any)
					if !ok {
						errCh <- mh.newMongoError(400, "unsupported data type in array")
						return
					}
					if mh.useTimestamps {
						if updated, ok := mh.appendTimestamps(itemMap, "insert").(map[string]any); ok {
							itemMap = updated
						} else {
							errCh <- mh.newMongoError(400, "failed to assert map after appending timestamps")
							return
						}
					}
					interfaceSlice = append(interfaceSlice, itemMap)
				}
				_, err := mh.collection.InsertMany(ctx, interfaceSlice)
				if err != nil {
					errCh <- mh.newMongoError(500, err.Error())
				} else {
					resultCh <- mh.newMongoOperations(200, true, "insert", "Insert performed.")
				}
			}(chunk)
		}
		wg.Wait()
		close(resultCh)
		close(errCh)
		for err := range errCh {
			if err.Error != "" {
				return MongoOperationsResult{}, err
			}
		}
		return mh.newMongoOperations(200, true, "insert", "All inserts performed."), MongoError{}
	default:
		return MongoOperationsResult{}, mh.newMongoError(400, "unsupported data type")
	}
	return mh.newMongoOperations(200, true, "insert", "Insert performed."), MongoError{}
}

// DropDatabase drops an entire database.
func (mh *MongoDBHandler) DropDatabase(dbName string) MongoError {
	if mh.client == nil {
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
	if err = mh.client.Database(dbName).Drop(ctx); err != nil {
		return mh.newMongoError(500, "Unable to drop database: "+dbName+". Error: "+err.Error())
	}
	return MongoError{}
}

// DropTable drops a specific collection from a database.
func (mh *MongoDBHandler) DropTable(dbName string, collectionName string) MongoError {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	databases, databasesErr := mh.ListDatabases()
	if databasesErr.Error != "" {
		return mh.newMongoError(500, fmt.Sprintf("Unable to retrieve database list: %s", databasesErr.Error))
	}

	dbExists := slices.Contains(databases.Databases, dbName)
	if !dbExists {
		return mh.newMongoError(404, fmt.Sprintf("Database not found: %s", dbName))
	}

	mh.db = mh.client.Database(dbName)
	tables, tablesErr := mh.ListCollections(dbName)
	if tablesErr.Error != "" {
		return mh.newMongoError(500, fmt.Sprintf("Unable to list collections: %s", tablesErr.Error))
	}

	if !slices.Contains(tables.Tables, collectionName) {
		return mh.newMongoError(404, fmt.Sprintf("Table/collection %s not found in database %s", collectionName, dbName))
	}

	mh.collection = mh.db.Collection(collectionName)
	if err := mh.collection.Drop(ctx); err != nil {
		return mh.newMongoError(500, fmt.Sprintf("Unable to drop collection %s: %s", collectionName, err.Error()))
	}

	return MongoError{}
}

// TotalCount returns the count of documents matching the current query.
func (mh *MongoDBHandler) TotalCount() (int64, error) {
	if mh.collection == nil {
		return 0, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	countOptions := options.Count().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2,
	})
	totalCount, err := mh.collection.CountDocuments(ctx, mh.query, countOptions)
	if err != nil {
		return 0, err
	}
	return totalCount, nil
}

func (mh *MongoDBHandler) AndAll(queryInput [][]any) *MongoDBHandler {
	for _, item := range queryInput {
		if len(item) >= 3 {
			field, ok1 := item[0].(string)
			operator, ok2 := item[1].(string)
			if ok1 && ok2 {
				mh.Where(field, operator, item[2])
			}
		}
	}
	return mh
}

func (mh *MongoDBHandler) OrAll(queryInput [][]any) *MongoDBHandler {
	for _, item := range queryInput {
		if len(item) >= 3 {
			field, ok1 := item[0].(string)
			operator, ok2 := item[1].(string)
			if ok1 && ok2 {
				mh.OrWhere(field, operator, item[2])
			}
		}
	}
	return mh
}

func (mh *MongoDBHandler) SortAll(sortInput [][]any) *MongoDBHandler {
	for _, item := range sortInput {
		if len(item) >= 2 {
			field, ok1 := item[0].(string)
			order, ok2 := item[1].(string)
			if ok1 && ok2 {
				mh.SortBy(field, order)
			}
		}
	}
	return mh
}

func (mh *MongoDBHandler) GroupAll(groupInput string) *MongoDBHandler {
	if groupInput != "" {
		mh.GroupBy(groupInput)
	}
	return mh
}

// ExecuteRaw runs either a simple Find or an Aggregate using raw JSON.
// aggregation is already supported, but, this does, indeed handle more stuff
// since the query is used sent
func (mh *MongoDBHandler) ExecuteRaw(rawJSON string, asPipeline bool) (MongoResults, MongoError) {
	// 1) ensure connection + collection is set
	if err := mh.getConnection(); err.Error != "" {
		return MongoResults{}, err
	}

	// 2) parse JSON into a go value
	var payload any
	if err := bson.UnmarshalExtJSON([]byte(rawJSON), true, &payload); err != nil {
		return MongoResults{}, mh.newMongoError(400, fmt.Sprintf("Invalid JSON: %s", err))
	}

	// 3) build find options for the non-pipeline branch
	findOpts := options.Find().
		SetLimit(int64(mh.perPage)).
		SetSkip(int64((mh.page - 1) * mh.perPage))
	if len(mh.sort) > 0 {
		findOpts.SetSort(mh.sort)
	}

	// 4) execute query
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	var (
		cur    *mongo.Cursor
		err    error
		filter any // for Find & CountDocuments
		docs   []map[string]any
	)

	if asPipeline {
		// normalize payload to []any
		var pipeline []any
		switch arr := payload.(type) {
		case []any:
			pipeline = append([]any{}, arr...) // copy
		case primitive.A:
			pipeline = make([]any, len(arr))
			copy(pipeline, arr)
		default:
			return MongoResults{}, mh.newMongoError(400, "For aggregation, JSON must be an array")
		}

		// inject $sort if provided
		if len(mh.sort) > 0 {
			pipeline = append(pipeline, bson.M{"$sort": mh.sort})
		}

		// ensure page ≥ 1
		if mh.page < 1 {
			mh.page = 1
		}

		// inject $skip and $limit
		pipeline = append(pipeline,
			bson.M{"$skip": int64((mh.page - 1) * mh.perPage)},
			bson.M{"$limit": int64(mh.perPage)},
		)

		cur, err = mh.collection.Aggregate(ctx, pipeline)
	} else {
		// accept document filter types
		switch f := payload.(type) {
		case bson.M, primitive.D, map[string]any:
			filter = f
		default:
			return MongoResults{}, mh.newMongoError(400, "For find, JSON must be an object")
		}
		cur, err = mh.collection.Find(ctx, filter, findOpts)
	}

	if err != nil {
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}

	defer func(cur *mongo.Cursor, ctx context.Context) {
		err := cur.Close(ctx)
		if err != nil {
			mh.logger.Println(err.Error())
		}
	}(cur, ctx)

	// 5) decode results
	for cur.Next(ctx) {
		var doc map[string]any
		if err := cur.Decode(&doc); err != nil {
			return MongoResults{}, mh.newMongoError(500, err.Error())
		}
		docs = append(docs, doc)
	}
	if err = cur.Err(); err != nil {
		return MongoResults{}, mh.newMongoError(500, err.Error())
	}

	// 6) count total matching docs
	var total int64
	if asPipeline {
		total = int64(len(docs))
	} else {
		total, err = mh.collection.CountDocuments(ctx, filter)
		if err != nil {
			return MongoResults{}, mh.newMongoError(500, err.Error())
		}
	}

	// getting the query in a readable format
	var buf bytes.Buffer
	var compactQ string
	if err := json.Compact(&buf, []byte(rawJSON)); err != nil {
		compactQ = strings.TrimSpace(rawJSON)
	} else {
		compactQ = buf.String()
	}

	// 7) wrap into MongoResults
	totalPages := int((total + int64(mh.perPage) - 1) / int64(mh.perPage))
	return MongoResults{
		Status:   true,
		Code:     200,
		Database: mh.dbName,
		Table:    mh.tableName,
		Count:    total,
		Results:  docs,
		Pagination: MongoResultPagination{
			TotalPages:  totalPages,
			CurrentPage: mh.page,
			NextPage:    minInt(totalPages, mh.page+1),
			PrevPage:    maxInt(1, mh.page-1),
			LastPage:    totalPages,
			PerPage:     mh.perPage,
		},
		Query: compactQ,
	}, MongoError{}
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
func maxInt(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// Find executes either a simple find query or an aggregate pipeline (if built) and applies pagination.
// keeping it here as the next approach could be buggyg
//func (mh *MongoDBHandler) Find() (MongoResults, MongoError) {
//	if mh.client == nil {
//		if err := mh.getConnection(); err.Error != "" {
//			return MongoResults{}, err
//		}
//	}
//
//	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
//	defer cancel()
//
//	// Validate pagination parameters.
//	if mh.page < 1 {
//		mh.page = 1
//	}
//	if mh.perPage <= 0 {
//		mh.perPage = 10
//	}
//
//	opts := options.Find().SetCollation(&options.Collation{
//		Locale:   "en",
//		Strength: 2,
//	})
//	if len(mh.sort) > 0 {
//		opts.SetSort(mh.sort)
//	}
//	opts.SetLimit(int64(mh.perPage))
//	opts.SetSkip(int64((mh.page - 1) * mh.perPage))
//
//	resultsChan := make(chan []map[string]any, 1)
//	errChan := make(chan error, 1)
//
//	go func() {
//		var cur *mongo.Cursor
//		var err error
//		if len(mh.aggregateQuery) > 0 {
//			mh.logger.Println("Running aggregate query...")
//			cur, err = mh.collection.Aggregate(ctx, mh.aggregateQuery)
//		} else {
//			mh.logger.Println("Running filter query...")
//			cur, err = mh.collection.Find(ctx, mh.query, opts)
//		}
//		if err != nil {
//			errChan <- err
//			return
//		}
//		defer func(cur *mongo.Cursor, ctx context.Context) {
//			err := cur.Close(ctx)
//			if err != nil {
//				mh.logger.Println("Error closing cursor")
//			}
//		}(cur, ctx)
//
//		var results []map[string]any
//		for cur.Next(ctx) {
//			var result map[string]any
//			if err := cur.Decode(&result); err != nil {
//				errChan <- err
//				return
//			}
//			results = append(results, result)
//		}
//		if err := cur.Err(); err != nil {
//			errChan <- err
//			return
//		}
//		resultsChan <- results
//	}()
//
//	select {
//	case err := <-errChan:
//		return MongoResults{}, mh.newMongoError(500, err.Error())
//	case <-time.After(5 * time.Second):
//		return MongoResults{}, mh.newMongoError(500, "Timeout while fetching results")
//	case results := <-resultsChan:
//		totalCount, err := mh.TotalCount()
//		if err != nil {
//			return MongoResults{}, mh.newMongoError(500, err.Error())
//		}
//		totalPages := (int(totalCount) + mh.perPage - 1) / mh.perPage
//		currentPage := mh.page
//		prevPage := 1
//		nextPage := 1
//		if currentPage > 1 {
//			prevPage = currentPage - 1
//		}
//		if currentPage < totalPages {
//			nextPage = currentPage + 1
//		}
//		plainQuery, _ := mh.Query()
//		return MongoResults{
//			Status:   true,
//			Code:     200,
//			Database: mh.dbName,
//			Table:    mh.tableName,
//			Count:    totalCount,
//			Results:  results,
//			Pagination: MongoResultPagination{
//				TotalPages:  totalPages,
//				CurrentPage: currentPage,
//				NextPage:    nextPage,
//				PrevPage:    prevPage,
//				LastPage:    totalPages,
//				PerPage:     mh.perPage,
//			},
//			Query: plainQuery,
//		}, MongoError{}
//	}
//}

func (mh *MongoDBHandler) Find() (MongoResults, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoResults{}, err
		}
	}

	// normalize pagination
	if mh.page < 1 {
		mh.page = 1
	}
	if mh.perPage <= 0 {
		mh.perPage = 10
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// ---------------------------------------------------
	// 1) Regular filtering
	// ---------------------------------------------------
	if len(mh.aggregateQuery) == 0 {
		opts := options.Find().
			SetCollation(&options.Collation{Locale: "en", Strength: 2}).
			SetSkip(int64((mh.page - 1) * mh.perPage)).
			SetLimit(int64(mh.perPage))
		if len(mh.sort) > 0 {
			opts.SetSort(mh.sort)
		}

		resultsCh := make(chan []map[string]any, 1)
		errCh := make(chan error, 1)

		go func() {
			cur, err := mh.collection.Find(ctx, mh.query, opts)
			if err != nil {
				errCh <- err
				return
			}
			defer func(cur *mongo.Cursor, ctx context.Context) {
				err := cur.Close(ctx)
				if err != nil {
					mh.logger.Println(err.Error())
				}
			}(cur, ctx)

			var docs []map[string]any
			for cur.Next(ctx) {
				var doc map[string]any
				if err := cur.Decode(&doc); err != nil {
					errCh <- err
					return
				}
				docs = append(docs, doc)
			}
			if err := cur.Err(); err != nil {
				errCh <- err
				return
			}
			resultsCh <- docs
		}()

		select {
		case err := <-errCh:
			return MongoResults{}, mh.newMongoError(500, err.Error())
		case <-ctx.Done():
			return MongoResults{}, mh.newMongoError(500, "Timeout while fetching results")
		case docs := <-resultsCh:
			totalCount, err := mh.TotalCount()
			if err != nil {
				return MongoResults{}, mh.newMongoError(500, err.Error())
			}
			totalPages := (int(totalCount) + mh.perPage - 1) / mh.perPage
			prev, next := 1, 1
			if mh.page > 1 {
				prev = mh.page - 1
			}
			if mh.page < totalPages {
				next = mh.page + 1
			}
			q, _ := mh.Query()

			return MongoResults{
				Status:   true,
				Code:     200,
				Database: mh.dbName,
				Table:    mh.tableName,
				Count:    totalCount,
				Results:  docs,
				Pagination: MongoResultPagination{
					TotalPages:  totalPages,
					CurrentPage: mh.page,
					NextPage:    next,
					PrevPage:    prev,
					LastPage:    totalPages,
					PerPage:     mh.perPage,
				},
				Query: q,
			}, MongoError{}
		}
	}

	// ---------------------------------------------------
	// 2) Grouped‐by branch: uses 2 goroutines -> count and match
	// ---------------------------------------------------
	// build count pipeline
	countPipe := []bson.M{}
	if len(mh.query) > 0 {
		countPipe = append(countPipe, bson.M{"$match": mh.query})
	}
	// mh.aggregateQuery already contains match, group, sort, skip, limit
	// but for counting we only need match + group + count:
	// extract group stage:
	groupStage := mh.aggregateQuery[0].(bson.M)

	for _, stage := range mh.aggregateQuery {
		if m, ok := stage.(bson.M); ok {
			if _, hasGroup := m["$group"]; hasGroup {
				groupStage = m
				break
			}
		}
	}
	countPipe = append(countPipe, groupStage, bson.M{"$count": "total"})

	type countResult struct {
		Total int64 `bson:"total"`
	}
	cntCh := make(chan countResult, 1)
	errCh := make(chan error, 2)

	go func() {
		cur, err := mh.collection.Aggregate(ctx, countPipe)
		if err != nil {
			errCh <- err
			return
		}
		defer func(cur *mongo.Cursor, ctx context.Context) {
			err := cur.Close(ctx)
			if err != nil {
				errCh <- err
			}
		}(cur, ctx)

		var cr countResult
		cr.Total = 0
		if cur.Next(ctx) {
			if err := cur.Decode(&cr); err != nil {
				errCh <- err
				return
			}
		}
		cntCh <- cr
	}()

	// build data pipeline (reuse mh.aggregateQuery)
	dataCh := make(chan []map[string]any, 1)
	go func() {
		cur, err := mh.collection.Aggregate(ctx, mh.aggregateQuery)
		if err != nil {
			errCh <- err
			return
		}
		defer func(cur *mongo.Cursor, ctx context.Context) {
			err := cur.Close(ctx)
			if err != nil {
				errCh <- err
			}
		}(cur, ctx)

		var buckets []map[string]any
		for cur.Next(ctx) {
			var b map[string]any
			if err := cur.Decode(&b); err != nil {
				errCh <- err
				return
			}
			buckets = append(buckets, b)
		}
		dataCh <- buckets
	}()

	// wait for both
	var (
		cntRes countResult
		data   []map[string]any
	)
	for i := 0; i < 2; i++ {
		select {
		case err := <-errCh:
			return MongoResults{}, mh.newMongoError(500, err.Error())
		case cr := <-cntCh:
			cntRes = cr
		case buckets := <-dataCh:
			data = buckets
		case <-ctx.Done():
			return MongoResults{}, mh.newMongoError(500, "Timeout while aggregating")
		}
	}

	totalGroups := cntRes.Total
	totalPages := int((totalGroups + int64(mh.perPage) - 1) / int64(mh.perPage))
	prev, next := 1, 1
	if mh.page > 1 {
		prev = mh.page - 1
	}
	if mh.page < totalPages {
		next = mh.page + 1
	}

	q, _ := mh.Query()

	return MongoResults{
		Status:   true,
		Code:     200,
		Database: mh.dbName,
		Table:    mh.tableName,
		Count:    totalGroups,
		Results:  data,
		Pagination: MongoResultPagination{
			TotalPages:  totalPages,
			CurrentPage: mh.page,
			NextPage:    next,
			PrevPage:    prev,
			LastPage:    totalPages,
			PerPage:     mh.perPage,
		},
		Query: q,
	}, MongoError{}
}

// Update performs an update based on the current query.
func (mh *MongoDBHandler) Update(data any) (MongoOperationsResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoOperationsResult{}, err
		}
	}
	if mh.useTimestamps {
		data = mh.appendTimestamps(data, "update")
	}
	update := bson.M{"$set": data}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	opts := options.Update().SetCollation(&options.Collation{
		Locale:   "en",
		Strength: 2,
	})
	_, err := mh.collection.UpdateMany(ctx, mh.query, update, opts)
	if err != nil {
		return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
	}
	return mh.newMongoOperations(200, true, "update", "Update performed"), MongoError{}
}

// newMongoOperations creates a new operations result.
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

// UpdateByID updates a document by its _id.
func (mh *MongoDBHandler) UpdateByID(recordId string, data any) (MongoOperationsResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoOperationsResult{}, err
		}
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return MongoOperationsResult{}, mh.newMongoError(400, "data must be a map[string]any")
	}
	update := bson.M{"$set": dataMap}
	if mh.useTimestamps {
		toJsonString, _ := ConvertMapToJsonOrdered(dataMap)
		update = bson.M{"$set": AppendUpdatedAtToJson(toJsonString)}
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
	_, err := mh.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return MongoOperationsResult{}, mh.newMongoError(500, err.Error())
	}
	return mh.newMongoOperations(200, true, "updateById", "Update performed"), MongoError{}
}

// FindById retrieves a document by its _id.
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
	// Try to treat recordId as an ObjectID.
	if objID, errObj := primitive.ObjectIDFromHex(recordId); errObj == nil {
		filter = bson.M{"_id": objID}
		err = mh.collection.FindOne(ctx, filter).Decode(&result)
		if err == nil {
			resultMap := make(map[string]any)
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
	// Otherwise, treat recordId as a string.
	filter = bson.M{"_id": recordId}
	err = mh.collection.FindOne(ctx, filter).Decode(&result)
	if err != nil {
		return SingleMongoResult{}, mh.newMongoError(404, "Record not found!")
	}
	resultMap := make(map[string]any)
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

// DeleteById removes a document by its _id.
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
	return mh.newMongoOperations(200, true, "deleteById", "Delete operation performed."), MongoError{}
}

// Delete performs deletion based on the current query.
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
		Strength: 2,
	})
	_, err := mh.collection.DeleteMany(ctx, mh.query, opts)
	if err != nil {
		return mh.newMongoOperations(500, false, "delete", "Error deleting: "+err.Error()), MongoError{}
	}
	return mh.newMongoOperations(200, false, "delete", "Document(s) deleted"), MongoError{}
}

// Query returns the current query filter as a JSON string.
func (mh *MongoDBHandler) Query() (string, error) {
	if len(mh.aggregateQuery) > 0 {
		jsonData, err := json.Marshal(mh.aggregateQuery)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	} else if len(mh.query) > 0 {
		jsonData, err := json.Marshal(mh.query)
		if err != nil {
			return "", err
		}
		return string(jsonData), nil
	}
	return "", errors.New("no query provided")
}

func (mh *MongoDBHandler) ListDatabases() (MongoDatabaseListResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return MongoDatabaseListResult{}, err
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	databases, err := mh.client.ListDatabaseNames(ctx, bson.D{})
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

func (mh *MongoDBHandler) Count() (CountMongoResult, MongoError) {
	if mh.client == nil {
		if err := mh.getConnection(); err.Error != "" {
			return CountMongoResult{}, err
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if len(mh.aggregateQuery) == 0 {
		count, err := mh.collection.CountDocuments(ctx, mh.query)
		if err != nil {
			return CountMongoResult{}, MongoError{Error: err.Error()}
		}
		q, _ := mh.Query()

		return CountMongoResult{
			Status:   true,
			Code:     200,
			Database: mh.dbName,
			Table:    mh.tableName,
			Count:    count,
			Message:  "Documents have been counted based on filters.",
			Query:    q,
		}, MongoError{}
	}

	var countPipe []bson.M
	if len(mh.query) > 0 {
		countPipe = append(countPipe, bson.M{"$match": mh.query})
	}

	for _, stage := range mh.aggregateQuery {
		m, ok := stage.(bson.M)
		if !ok {
			continue
		}
		if _, isSkip := m["$skip"]; isSkip {
			continue
		}
		if _, isLimit := m["$limit"]; isLimit {
			continue
		}
		if _, isSort := m["$sort"]; isSort {
			continue
		}
		countPipe = append(countPipe, m)
	}
	// Append the final count stage
	countPipe = append(countPipe, bson.M{"$count": "total"})

	cursor, err := mh.collection.Aggregate(ctx, countPipe)
	if err != nil {
		return CountMongoResult{}, MongoError{Error: err.Error()}
	}
	defer func(cur *mongo.Cursor) {
		if err := cur.Close(ctx); err != nil {
			mh.logger.Println(err.Error())
		}
	}(cursor)

	type countResult struct {
		Total int64 `bson:"total"`
	}
	var cr countResult
	if cursor.Next(ctx) {
		if err := cursor.Decode(&cr); err != nil {
			return CountMongoResult{}, MongoError{Error: err.Error()}
		}
	}

	if err := cursor.Err(); err != nil {
		return CountMongoResult{}, MongoError{Error: err.Error()}
	}

	pipeJSON, err := json.Marshal(countPipe)
	if err != nil {
		return CountMongoResult{}, MongoError{Error: err.Error()}
	}

	return CountMongoResult{
		Status:   true,
		Code:     200,
		Database: mh.dbName,
		Table:    mh.tableName,
		Count:    cr.Total,
		Message:  "Documents have been counted based on aggregation filters.",
		Query:    string(pipeJSON),
	}, MongoError{}
}

// ResetQuery clears the current query filter and aggregation pipeline.
func (mh *MongoDBHandler) ResetQuery() *MongoDBHandler {
	mh.aggregateQuery = []any{}
	mh.query = make(map[string]any)
	mh.multipleWheres = false
	return mh
}

// ResetSort clears any sorting settings.
func (mh *MongoDBHandler) ResetSort() *MongoDBHandler {
	mh.sort = []primitive.E{}
	return mh
}

// ResetState clears the current query and sort but leaves the DB/table names intact.
func (mh *MongoDBHandler) ResetState() *MongoDBHandler {
	mh.query = make(map[string]any)
	mh.sort = []primitive.E{}
	return mh
}

func (mh *MongoDBHandler) newMongoError(code int, errMsg string) MongoError {
	queryStr, _ := mh.Query()
	return MongoError{
		Status:   false,
		Code:     code,
		Database: mh.dbName,
		Table:    mh.tableName,
		Error:    errMsg,
		Query:    queryStr,
	}
}
