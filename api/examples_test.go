package main

import (
	"fmt"
	"log"
)

func ExampleGetObjectFromS3() {
	bytes, err := getFileFromS3("mybucket-gagagagagagag-2020", "path/to/my/key")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println("object: ", string(bytes))
	// Output:
	// asd
}