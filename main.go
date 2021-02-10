package main

import (
	"encoding/json"
	"fmt"
	"reflect"
)

// Sample - пример
type Sample struct {
	ID       int
	Username string
	Active   bool
}

func main() {
	expected := &Sample{
		ID:       42,
		Username: "rvasily",
		Active:   true,
	}
	jsonRaw, _ := json.Marshal(expected)
	// fmt.Println(string(jsonRaw))

	var tmpData interface{}
	json.Unmarshal(jsonRaw, &tmpData)

	result := new(Sample)
	err := i2s(tmpData, result)

	if err != nil {
		fmt.Printf("unexpected error: %v", err)
	}
	if !reflect.DeepEqual(expected, result) {
		fmt.Printf("results not match\nGot:\n%#v\nExpected:\n%#v", result, expected)
	}
}
