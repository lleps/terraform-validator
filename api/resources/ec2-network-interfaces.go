package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2NetworkInterface struct {
	svc *ec2.EC2
	eni *ec2.NetworkInterface
}

func init() {
	register("EC2NetworkInterface", ListEC2NetworkInterfaces)
}

func ListEC2NetworkInterfaces(sess *session.Session) ([]Resource, error) {
	svc := ec2.New(sess)

	resp, err := svc.DescribeNetworkInterfaces(nil)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, out := range resp.NetworkInterfaces {

		resources = append(resources, &EC2NetworkInterface{
			svc: svc,
			eni: out,
		})
	}

	return resources, nil
}

func (e *EC2NetworkInterface) ID() string {
	return *e.eni.NetworkInterfaceId
}

func (e *EC2NetworkInterface) Details() string {
	return e.eni.String()
}
