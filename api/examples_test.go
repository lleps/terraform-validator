package main

import (
	"api/resources"
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

func ExampleSendValidationToSlack() {
	slackWebHookUrl := "https://hooks.slack.com/services/YOUR_EXAMPLE_SLACK_KEYS_HERE"
	err := reportFailedValidationToSlack(
		slackWebHookUrl,
		"test.com",
		&TFState{
			Id:     "my-tfstate-id",
			Bucket: "test-bucket",
			Path:   "/some/path",
		})
	if err != nil {
		panic(err)
	}
	// Output:
}

func ExampleListAllResources() {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String("us-west-2")},
	)
	if err != nil {
		panic(err)
	}
	rs, err := resources.ListAllResources(sess)
	if err != nil {
		panic(err)
	}

	fmt.Println("Resources: ")
	for _, res := range rs {
		fmt.Println(res.Resource.ID())
	}
	fmt.Println("Ok.")
	// Output:
	// This is just runnable. But if I don't specify
	// output comment GoLand doesn't let me run this.
}
