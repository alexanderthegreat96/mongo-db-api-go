package main

import (
	"log"
	"os"
	"strconv"
	"time"

	"github.com/alexanderthegreat96/mongo-db-api-go/api"
	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"github.com/common-nighthawk/go-figure"
	"github.com/joho/godotenv"
)

var apiKey string
var apiPort string
var apiHost string
var canBoot bool
var shoudlWaitForMongoOnBoot bool
var logger *log.Logger
var mongoDb *driver.MongoDBHandler

const maxRetries = 5
const retryInterval = 5 * time.Second

func init() {
	logger = log.New(os.Stdout, "[MONGO-API]: ", log.Ldate|log.Ltime)

	versionNumber := "v1.0.5"
	mongoApiBanner := figure.NewColorFigure("MongoAPI "+versionNumber, "", "blue", false)
	mongoApiBanner.Print()

	canBoot = true
	err := godotenv.Load()
	if err != nil {
		canBoot = false
		logger.Println("Unable to load .env file.")
		return
	}

	logger.Println("Using MongoDB Server Information:")
	logger.Printf("MongoDB Host: %s", helpers.GetEnv("MONGO_DB_HOST", "Missing: MONGO_DB_HOST"))
	logger.Printf("MongoDB Port: %s", helpers.GetEnv("MONGO_DB_PORT", "Missing: MONGO_DB_PORT"))
	logger.Printf("MongoDB Username: %s", helpers.GetEnv("MONGO_DB_USERNAME", "Missing: MONGO_DB_USERNAME"))
	logger.Printf("MongoDB Password: %s", helpers.GetEnv("MONGO_DB_PASSWORD", "Missing: MONGO_DB_PASSWORD"))
	logger.Printf("MongoDB Default Database: %s", helpers.GetEnv("MONGO_DB_NAME", "Missing: MONGO_DB_NAME"))
	logger.Printf("MongoDB Default Table: %s", helpers.GetEnv("MONGO_DB_TABLE", "Missing: MONGO_DB_TABLE"))

	shoudlWaitForMongoOnBoot, _ = strconv.ParseBool(helpers.GetEnv("WAIT_FOR_MONGO_ON_BOOT", "false"))
	if shoudlWaitForMongoOnBoot {
		if !waitForMongoConnection() {
			canBoot = false
			logger.Println("Unable to connect to the MongoDB server after multiple attempts. Exiting.")
			return
		}
	}

	apiKey = helpers.GetEnv("API_KEY", "")
	apiPort = helpers.GetEnv("API_PORT", "9776")
	apiHost = helpers.GetEnv("API_HOST", "localhost")
	mongoDb = driver.MongoDB()
}

func waitForMongoConnection() bool {
	for i := 1; i <= maxRetries; i++ {
		if driver.MongoDB().CanConnectToMongo() {
			logger.Println("Successfully connected to the MongoDB server.")
			return true
		}

		logger.Printf("Attempt %d/%d: Unable to connect to the MongoDB server. Retrying in %v...\n", i, maxRetries, retryInterval)
		time.Sleep(retryInterval)
	}

	return false
}

func main() {
	if !canBoot {
		logger.Println("Errors are present. Unable to boot the API.")
		return
	}

	logger.Println("API Information:")
	logger.Println("You may start sending requests to: http://" + apiHost + ":" + apiPort)

	api.RunApi(*mongoDb, apiKey, apiHost, apiPort)
}
