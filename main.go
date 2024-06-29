package main

import (
	"log"
	"os"

	"github.com/alexanderthegreat96/mongo-db-api-go/api"
	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"github.com/common-nighthawk/go-figure"
	"github.com/joho/godotenv"
)

var apiPort string
var apiHost string
var canBoot bool
var logger *log.Logger
var mongoDb *driver.MongoDBHandler

func init() {
	logger = log.New(os.Stdout, "[MONGO-API]: ", log.Ldate|log.Ltime)

	versionNumber := "v1.0"
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

	canConnect := driver.MongoDB().CanConnectToMongo()

	if !canConnect {
		canBoot = false
		logger.Println("Unable to connect to the MongoDB server. Please check your settings.")
		return
	}

	apiPort = helpers.GetEnv("API_PORT", "9776")
	apiHost = helpers.GetEnv("API_HOST", "localhost")
	mongoDb = driver.MongoDB()
}

func main() {
	if !canBoot {
		logger.Println("Errors are present. Unable to boot the API.")
		return
	}

	logger.Println("API Information:")
	logger.Println("You may start sending requests to: http://" + apiHost + ":" + apiPort)

	// boot the actual API
	api.RunApi(*mongoDb, apiHost, apiPort)
}
