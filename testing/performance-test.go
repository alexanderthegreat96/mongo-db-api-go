package main

import (
	"fmt"
	"time"

	"github.com/alexanderthegreat96/mongo-db-api-go/driver"
	"github.com/joho/godotenv"
)

func main() {
	godotenv.Load("../.env")

	// generates 5kkk records
	data := driver.GenerateRandomData(5000000)
	start := time.Now()

	// should result in about 4.92 seconds
	mh := driver.MongoDB()
	result, err := mh.Insert(data)
	if err.Error != "" {
		fmt.Println("Error:", err.Error)
	} else {
		fmt.Println("Result:", result)
	}
	elapsed := time.Since(start)

	fmt.Printf("Insertion took %s\n", elapsed)
}
