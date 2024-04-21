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

	// record := map[string]interface{}{
	// 	"firstName": "Matthew",
	// 	"lastName":  "Something",
	// 	"age":       21,
	// }

	data, err := driver.
		NewMongoHandler().
		FromDB("alexanderdth").
		FromTable("my-table").
		Where("firstName", "!=", "HfEOTJ").
		PerPage(10).
		Page(50).
		Find()

	if err.Error != "" {
		// Handle the error
		fmt.Println("Error:", err)
	} else {
		// Print the data
		fmt.Println("Status:", data.Status)
		fmt.Println("Code:", data.Code)
		fmt.Println("Database:", data.Database)
		fmt.Println("Table:", data.Table)
		fmt.Println("Count:", data.Count)
		fmt.Println("Pagination:")
		fmt.Println("\tTotalPages:", data.Pagination.TotalPages)
		fmt.Println("\tCurrentPage:", data.Pagination.CurrentPage)
		fmt.Println("\tNextPage:", data.Pagination.NextPage)
		fmt.Println("\tLastPage:", data.Pagination.LastPage)
		fmt.Println("\tPerPage:", data.Pagination.PerPage)

	}

	// rand.Seed(time.Now().UnixNano())

	// // Create a slice to store the records
	// var records []map[string]interface{}

	// // Generate 40 records with random data
	// for i := 0; i < 500; i++ {
	// 	record := make(map[string]interface{})
	// 	record["firstName"] = randomString(6)
	// 	record["lastName"] = randomString(8)
	// 	record["age"] = rand.Intn(50) + 18 // Random age between 18 and 67
	// 	records = append(records, record)
	// }

	// err := driver.NewMongoHandler().IntoDB("alexanderdth").IntoTable("my-table").Insert(record)

	// if err != nil {
	// 	fmt.Println(err)
	// }
}

// func randomString(length int) string {
// 	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
// 	b := make([]byte, length)
// 	for i := range b {
// 		b[i] = charset[rand.Intn(len(charset))]
// 	}
// 	return string(b)
// }
