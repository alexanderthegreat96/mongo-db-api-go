package main

import (
	"fmt"
)

// CreateOrderedMap adds key-value pairs to a map and maintains the insertion order
func CreateOrderedMap(m map[interface{}]interface{}, keyOrder *[]interface{}, key, value interface{}) {
	// Initialize the map and keyOrder slice if they are nil
	if m == nil {
		m = make(map[interface{}]interface{})
	}
	if keyOrder == nil {
		*keyOrder = []interface{}{}
	}

	// Only add the key to the keyOrder slice if it doesn't already exist
	if _, exists := m[key]; !exists {
		*keyOrder = append(*keyOrder, key)
	}
	m[key] = value
}

func main() {
	// Initialize the map and keyOrder slice
	resultMap := make(map[interface{}]interface{})
	var keyOrder []interface{}

	// Add key-value pairs
	CreateOrderedMap(resultMap, &keyOrder, "username", "someone")
	CreateOrderedMap(resultMap, &keyOrder, "password", "123123")
	CreateOrderedMap(resultMap, &keyOrder, "dob", "random")
	CreateOrderedMap(resultMap, &keyOrder, "created_at", "today")
	CreateOrderedMap(resultMap, &keyOrder, "updated_At", "today")

	// Print the ordered map
	fmt.Println("Original ordered map:")
	fmt.Println("{")
	for _, key := range keyOrder {
		fmt.Printf("  %v: %v\n", key, resultMap[key])
	}
	fmt.Println("}")

	// Create a new map with the same order
	new := make(map[interface{}]interface{})
	for _, key := range keyOrder {
		new[key] = resultMap[key]
	}

	// Print the new map with the same order
	fmt.Println("\nNew ordered map:")

	fmt.Println(new)
	fmt.Println("{")
	for _, key := range keyOrder {
		fmt.Printf("  %v: %v\n", key, new[key])
	}
	fmt.Println("}")
}
