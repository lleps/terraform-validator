package resources

import (
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
	"strconv"
	"strings"
)

type EC2Image struct {
	svc  *ec2.EC2
	id   string
	obj  *ec2.Image
	tags []*ec2.Tag
}

func init() {
	register("EC2Image", ListEC2Images)
}

func ListEC2Images(sess *session.Session) ([]Resource, error) {
	svc := ec2.New(sess)
	params := &ec2.DescribeImagesInput{
		Owners: []*string{
			aws.String("self"),
		},
	}
	resp, err := svc.DescribeImages(params)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, out := range resp.Images {
		resources = append(resources, &EC2Image{
			svc:  svc,
			obj:  out,
			id:   *out.ImageId,
			tags: out.Tags,
		})
	}

	return resources, nil
}

func (e *EC2Image) Remove() error {
	_, err := e.svc.DeregisterImage(&ec2.DeregisterImageInput{
		ImageId: &e.id,
	})
	return err
}

func (e *EC2Image) Details() string {
	sb := strings.Builder{}
	sb.WriteString("Tags:\n")
	for i, tagValue := range e.tags {
		sb.WriteString("#" + strconv.Itoa(i) + ": " + *tagValue.Key + " => " + *tagValue.Value)
		sb.WriteRune('\n')
	}
	return sb.String()
}

func (e *EC2Image) ID() string {
	return e.id
}
