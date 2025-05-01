package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/alexanderthegreat96/envparser/v2"
	"github.com/alexanderthegreat96/mongo-db-api-go/api"
	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/common-nighthawk/go-figure"
)

var apiKey string
var apiPort string
var apiHost string
var canBoot bool
var logger *log.Logger
var mongoDb *driver.MongoDBHandler

const maxRetries = 5
const retryInterval = 5 * time.Second

func init() {
	logger = log.New(os.Stdout, "[MONGO-API]: ", log.Ldate|log.Ltime)
	env := envparser.NewEnvParser(envparser.WithFilename(".env"), envparser.WithRootPath(true))

	versionNumber := "v1.0.6"
	mongoApiBanner := figure.NewColorFigure(fmt.Sprintf("MongoAPI %s", versionNumber), "", "blue", false)
	mongoApiBanner.Print()
	fmt.Println()

	canBoot = true
	if env.GetError() != "" {
		canBoot = false
		logger.Printf("Issue loading .env file. Err: %s", env.GetError())
		return
	}

	// used for config checking
	rawHost, _ := env.GetValue("MONGO_DB_HOST", "string", "localhost")
	rawPort, _ := env.GetValue("MONGO_DB_PORT", "string", "27017")
	rawUser, _ := env.GetValue("MONGO_DB_USERNAME", "string", "admin")
	rawPass, _ := env.GetValue("MONGO_DB_PASSWORD", "string", "admin")
	rawDB, _ := env.GetValue("MONGO_DB_NAME", "string", "test")
	rawTable, _ := env.GetValue("MONGO_DB_TABLE", "string", "test")
	rawWaitBoot, _ := env.GetValue("WAIT_FOR_MONGO_ON_BOOT", "bool", false)
	rawWaitSecs, _ := env.GetValue("WAIT_AT_BOOT", "int", 30)

	// used for initalizing the API
	rawAPIKey, _ := env.GetValue("API_KEY", "string", "")
	rawAPIHost, _ := env.GetValue("API_HOST", "string", "0.0.0.0")
	rawAPIPort, _ := env.GetValue("API_PORT", "string", "9776")

	mongoHost := rawHost.(string)
	mongoPort := rawPort.(string)
	mongoUser := rawUser.(string)
	mongoPass := rawPass.(string)
	mongoDatabase := rawDB.(string)
	mongoCollection := rawTable.(string)
	waitForMongo := rawWaitBoot.(bool)
	waitSeconds := rawWaitSecs.(int)
	apiKey = rawAPIKey.(string)
	apiHost = rawAPIHost.(string)
	apiPort = rawAPIPort.(string)

	logger.Println("Using MongoDB Server Information:")
	logger.Printf("MongoDB Host: %s", mongoHost)
	logger.Printf("MongoDB Port: %s", mongoPort)
	logger.Printf("MongoDB Username: %s", mongoUser)
	logger.Printf("MongoDB Password: %s", mongoPass)
	logger.Printf("MongoDB Default Database: %s", mongoDatabase)
	logger.Printf("MongoDB Default Collection: %s", mongoCollection)
	logger.Printf("Should wait for MongoDB to boot up: %t", waitForMongo)
	logger.Printf("Wait time for MongoDB to boop up: %d", waitSeconds)

	if apiKey != "" {
		logger.Printf("Requires API Key: %s", apiKey)
	}

	logger.Printf("API Host: %s", apiHost)
	logger.Printf("API Port: %s", apiPort)

	if waitForMongo {
		if waitSeconds > 0 {
			logger.Printf("Waiting %d seconds before attempting MongoDB connection...\n", waitSeconds)
			for i := waitSeconds; i > 0; i-- {
				fmt.Printf("\r⏳ Connecting in %2d seconds... ", i)
				time.Sleep(1 * time.Second)
			}
			fmt.Println("\r🚀 Attempting to connect now...         ")
		}
	}

	if !waitForMongoConnection() {
		canBoot = false
		logger.Println("Unable to connect to the MongoDB server after multiple attempts. Exiting.")
		return
	}

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
