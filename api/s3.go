// This file provides some methods to fetch s3 bucket contents in one line.
package main

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"time"
)

// getItemFromS3IfChanged fetches the content of the given bucket-item as string only
// if the item last modification date is different from the passed prevLastUpdate.
// Otherwise, sets the changed bool to false and returns an empty string.
func getItemFromS3IfChanged(
	sess *session.Session,
	bucket string,
	item string,
	prevLastModification string,
) (changed bool, content []byte, lastModification string, err error) {
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

	// Check if changed. Return now if didn't. Also set return value lastModification in any case.
	timeFormat := time.ANSIC
	lastModification = head.LastModified.Format(timeFormat)
	changed = lastModification != prevLastModification
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

	content = buf.Bytes()
	return
}
