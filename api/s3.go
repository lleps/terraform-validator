package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"time"
)

// getStringFromS3IfChanged fetches the content of the given bucket and item only
// if the item last modification date is different from the given lastUpdate.
// Otherwise, sets the changed bool to false and returns nothing.
func getStringFromS3IfChanged(
	sess *session.Session,
	bucket string,
	item string,
	prevLastUpdate string,
) (changed bool, content string, lastUpdate string, err error) {
	svc := s3.New(sess)

	// Get object head
	head, err := svc.HeadObject(&s3.HeadObjectInput{
		Bucket: aws.String(bucket),
		Key:    aws.String(item),
	})
	if err != nil {
		err = fmt.Errorf("can't get object head data for %s:%s: %v", bucket, item, err)
		return
	}

	// Check if changed. Return now if didn't. Also set return value lastUpdate in any case.
	timeFormat := time.ANSIC
	lastUpdate = head.LastModified.Format(timeFormat)
	changed = lastUpdate != prevLastUpdate
	if !changed {
		return
	}

	// Download
	downloader := s3manager.NewDownloader(sess)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err = downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})

	if err != nil {
		err = fmt.Errorf("can't download item for %s:%s: %v", bucket, item, err)
		return
	}

	content = string(buf.Bytes())
	return
}

func getFileFromS3(bucket, item string) ([]byte, error) {
	sess, _ := session.NewSession(&aws.Config{
		Region: aws.String("us-east-1")},
	)
	downloader := s3manager.NewDownloader(sess)
	buf := aws.NewWriteAtBuffer([]byte{})
	_, err := downloader.Download(buf,
		&s3.GetObjectInput{
			Bucket: aws.String(bucket),
			Key:    aws.String(item),
		})

	if err != nil {
		return nil, fmt.Errorf("can't download item: '%s': %v", item, err)
	}

	return buf.Bytes(), nil
}
