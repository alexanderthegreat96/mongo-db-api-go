package main

import (
	"fmt"

	"github.com/alexanderthegreat96/mongo-db-api-go/helpers"
)

func main() {

	user := map[string]interface{}{
		"username": "alex",
		"password": "1231",
		"age":      88,
		"prof":     []string{"welder", "mechanic"},
	}

	fmt.Println(helpers.ConvertMapToJsonOrdered(user))

}
