package resources

import (
	"strconv"
	"strings"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
)

func init() {
	register("S3Bucket", ListS3Buckets)
}

type S3Bucket struct {
	svc  *s3.S3
	name string
	tags []*s3.Tag
}

func ListS3Buckets(s *session.Session) ([]Resource, error) {
	svc := s3.New(s)

	buckets, err := DescribeS3Buckets(svc)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, name := range buckets {
		tags, err := svc.GetBucketTagging(&s3.GetBucketTaggingInput{
			Bucket: aws.String(name),
		})

		if err != nil {
			if aerr, ok := err.(awserr.Error); ok {
				if aerr.Code() == "NoSuchTagSet" {
					resources = append(resources, &S3Bucket{
						svc:  svc,
						name: name,
						tags: make([]*s3.Tag, 0),
					})
				}
			}
			continue
		}

		resources = append(resources, &S3Bucket{
			svc:  svc,
			name: name,
			tags: tags.TagSet,
		})
	}

	return resources, nil
}

func DescribeS3Buckets(svc *s3.S3) ([]string, error) {
	resp, err := svc.ListBuckets(nil)
	if err != nil {
		return nil, err
	}

	buckets := make([]string, 0)
	for _, out := range resp.Buckets {
		buckets = append(buckets, *out.Name)
	}

	return buckets, nil
}

func (e *S3Bucket) Details() string {
	sb := strings.Builder{}
	sb.WriteString("Tags:\n")
	for i, tagValue := range e.tags {
		sb.WriteString("#" + strconv.Itoa(i) + ": " + *tagValue.Key + " => " + *tagValue.Value)
		sb.WriteRune('\n')
	}
	return sb.String()
}

func (e *S3Bucket) ID() string {
	return e.name
}
