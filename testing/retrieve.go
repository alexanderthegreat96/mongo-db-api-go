package main

import (
	"fmt"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	result, err := driver.MongoDB().DB("test").
		Table("test").
		Where("age", "between", []interface{}{16, 18}).
		Where("name", "ilike", "david").
		SortBy("date", "desc").
		GroupBy("name").
		Find()

	if err.Error != "" {
		fmt.Println(err.Error)
	} else {
		fmt.Println(result.Query)
	}
}
