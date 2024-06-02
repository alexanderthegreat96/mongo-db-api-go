package main

import (
	"fmt"
	"log"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/joho/godotenv"
)

func main() {

	envError := godotenv.Load()
	if envError != nil {
		log.Fatal("Error loading .env file")
	}

	data, err := driver.
		MongoDB().
		DB("movie-dataset").
		Table("genome-scores").
		FindById("665b6fb9c68a61c818accfaf")

	fmt.Println(data)
	fmt.Println(err)

	// if err.Error != "" {
	// 	// Handle the error
	// 	fmt.Println("Error:", err.Error)
	// 	fmt.Println("Used Query:", err.Query)
	// } else {
	// 	fmt.Println(data.Query)
	// 	// Print the data
	// 	fmt.Println("Status:", data.Status)
	// 	fmt.Println("Code:", data.Code)
	// 	fmt.Println("Database:", data.Database)
	// 	fmt.Println("Table:", data.Table)
	// 	fmt.Println("Count:", data.Count)
	// 	fmt.Println("Pagination:")
	// 	fmt.Println("\tTotalPages:", data.Pagination.TotalPages)
	// 	fmt.Println("\tCurrentPage:", data.Pagination.CurrentPage)
	// 	fmt.Println("\tNextPage:", data.Pagination.NextPage)
	// 	fmt.Println("\tLastPage:", data.Pagination.LastPage)
	// 	fmt.Println("\tPerPage:", data.Pagination.PerPage)
	// }

	// dbs, dbsErr := driver.MongoDB().ListDatabases()
	// fmt.Println(dbs)
	// fmt.Println(dbsErr)

	// fmt.Println(driver.MongoDB().DropDatabase("delete-this-db"))

	// fmt.Println(helpers.ParseQuery("[username,!=,alexanderdth|age,>,24]"))
	// fmt.Println(helpers.ParseSort("[created_at:desc|first_name:asc]"))
}
