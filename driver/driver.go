package driver

import (
	"context"
	"strconv"
	"time"

	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

type MongoHandler struct {
	debug         bool
	limit         int64
	perPage       int64
	page          int64
	sort          []string
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

func NewMongoHandler() *MongoHandler {
	host := helpers.GetEnv("MONGO_DB_HOST", "localhost")
	port := helpers.GetEnv("MONGO_DB_PORT", "27017")
	dbName := helpers.GetEnv("MONGO_DB_NAME=", "test")
	tableName := helpers.GetEnv("MONGO_DB_TABLE", "test")
	username := helpers.GetEnv("MONGO_DB_USERNAME", "admin")
	password := helpers.GetEnv("MONGO_DB_PASSWORD", "admin")
	useTimestamps, _ := strconv.ParseBool(helpers.GetEnv("HANDLER_USE_TIMESTAMPS", "true"))

	return &MongoHandler{
		debug:         false,
		limit:         0,
		perPage:       10,
		page:          1,
		sort:          []string{},
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

func (mh *MongoHandler) FromTable(tableName string) *MongoHandler {
	mh.tableName = tableName
	return mh
}

func (mh *MongoHandler) IntoTable(tableName string) *MongoHandler {
	mh.tableName = tableName
	return mh
}

func (mh *MongoHandler) FromDB(dbName string) *MongoHandler {
	mh.dbName = dbName
	return mh
}

func (mh *MongoHandler) IntoDB(dbName string) *MongoHandler {
	mh.dbName = dbName
	return mh
}

// where contraints mapped
func (mh *MongoHandler) Where(field string, operator string, value interface{}) *MongoHandler {
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
		mh.query[field] = operatorMap[operator]
	}

	return mh
}

func (mh *MongoHandler) AndWhere(field, operator string, value interface{}) *MongoHandler {
	operatorMap := map[string]interface{}{
		"=":     value,
		"!=":    map[string]interface{}{"$ne": value},
		"<":     map[string]interface{}{"$lt": value},
		"<=":    map[string]interface{}{"$lte": value},
		">":     map[string]interface{}{"$gt": value},
		">=":    map[string]interface{}{"$gte": value},
		"_like": map[string]interface{}{"$regex": value, "options": "i"},
	}

	and_condition := make(map[string]interface{})

	if _, ok := operatorMap[operator]; ok {
		and_condition[field] = operatorMap[operator]
	}

	// check for $and contraints in the query
	if _, ok := mh.query["$and"]; !ok {
		mh.query["$and"] = []interface{}{}
	}

	mh.query["$and"] = append(mh.query["$and"].([]interface{}), and_condition)

	return mh
}

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
		data = mh.appendTimestamps(data)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// check if it is slice
	// of maps
	if _, ok := data.([]interface{}); ok {
		_, err := mh.collection.InsertMany(ctx, data.([]interface{}))
		if err != nil {
			return err
		}
	} else {
		// just insert one
		_, err := mh.collection.InsertOne(ctx, data)
		if err != nil {
			return err
		}
	}

	return nil
}

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

// some type of query builder
func (mh *MongoHandler) Find(query interface{}) ([]map[string]interface{}, error) {
	if err := mh.getConnection(); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	cur, err := mh.collection.Find(ctx, query)
	if err != nil {
		return nil, err
	}
	defer cur.Close(ctx)

	var results []map[string]interface{}
	for cur.Next(ctx) {
		var result map[string]interface{}
		if err := cur.Decode(&result); err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	if err := cur.Err(); err != nil {
		return nil, err
	}

	return results, nil
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
