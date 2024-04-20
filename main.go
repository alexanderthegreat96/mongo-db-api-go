package main

import (
	"fmt"
	"log"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
	"github.com/joho/godotenv"
)

var apiPort string

func init() {
	// ensure loading .env file in the
	// root of the probam
	// otherwise driver cannot acccess
	// it's contents

	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	apiPort = helpers.GetEnv("API_PORT", "9776")
}

func main() {
	fmt.Println(apiPort)

	// jsonData := map[string]interface{}{
	// 	"firstName": "Matthew",
	// 	"lastName":  "Something",
	// 	"age":       21,
	// }

	err := driver.
		NewMongoHandler().
		FromDB("alexanderdth").
		FromTable("my-table").
		AndWhere("alex", "!=", "someone").
		AndWhere("age", "!=", "someone")

	fmt.Println(err)

}
