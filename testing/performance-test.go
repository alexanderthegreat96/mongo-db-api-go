package main

import (
	"fmt"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	// generates 5kkk records
	data := []map[string]interface{}{
		{
			"username": "someone",
			"password": 123123,
			"dob":      "random",
		},
		{
			"username": "another",
			"password": 456456,
			"dob":      "different",
		},
	}
	// should result in about 4.92 seconds
	mh := driver.MongoDB()
	result, err := mh.Insert(data)
	if err.Error != "" {
		fmt.Println("Error:", err.Error)
	} else {
		fmt.Println("Result:", result)
	}
	// elapsed := time.Since(start)

	// fmt.Printf("Insertion took %s\n", elapsed)
}
