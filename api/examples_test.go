package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"log"
)

func ExampleGetItemFromS3IfChanged() {
	sess := session.Must(session.NewSession())
	changed, content, lastModification, err := getItemFromS3IfChanged(sess, "mybucket-gagagagagagag-2020", "path/to/my/key", "")
	if err != nil {
		log.Fatalf("error: %v", err)
	}
	fmt.Println("object: ", string(content))
	fmt.Println("changed: ", changed)
	fmt.Println("last modification: ", lastModification)
	// Output:
	// This is just runnable. But if I don't specify
	// output comment GoLand doesn't let me run this.
}

func ExampleListAllResources() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}
	resources, err := ListAllResources(sess)
	if err != nil {
		panic(err)
	}

	fmt.Println("Resources: ")
	for _, res := range resources {
		fmt.Println(res.ID())
	}
	fmt.Println("Ok.")
	// Output:
	// This is just runnable. But if I don't specify
	// output comment GoLand doesn't let me run this.
}
