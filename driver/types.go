package driver

import (
	"go.mongodb.org/mongo-driver/bson"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/bson/primitive"
	"go.mongodb.org/mongo-driver/mongo"
)

type MongoDBHandler struct {
	debug          bool
	limit          int
	perPage        int
	innerPage      int
	innerPerPage   int
	page           int
	sort           []primitive.E
	query          map[string]any
	aggregateQuery bson.A
	multipleWheres bool
	host           string
	port           string
	dbName         string
	tableName      string
	username       string
	password       string
	useTimestamps  bool
	timeNow        time.Time
	client         *mongo.Client
	db             *mongo.Database
	collection     *mongo.Collection
	logger         *log.Logger
}

// Result and error types
type MongoResultPagination struct {
	TotalPages  int
	CurrentPage int
	NextPage    int
	PrevPage    int
	LastPage    int
	PerPage     int
}

type MongoResults struct {
	Status     bool
	Code       int
	Database   string
	Table      string
	Count      int64
	Results    []map[string]any
	Pagination MongoResultPagination
	Query      string
}

// MongoError represents errors that occur during operations.
type MongoError struct {
	Status   bool
	Code     int
	Database string
	Table    string
	Error    string
	Query    string // Added field for the query string
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
	Result   any
}

type CountMongoResult struct {
	Status   bool
	Code     int
	Database string
	Table    string
	Count    int64
	Message  string
	Query    string
}
