package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Address struct {
	svc *ec2.EC2
	id  string
	ip  string
	obj *ec2.Address
}

func init() {
	register("EC2Address", ListEC2Addresses)
}

func ListEC2Addresses(sess *session.Session) ([]Resource, error) {
	svc := ec2.New(sess)

	params := &ec2.DescribeAddressesInput{}
	resp, err := svc.DescribeAddresses(params)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, out := range resp.Addresses {
		resources = append(resources, &EC2Address{
			svc: svc,
			obj: out,
			id:  *out.AllocationId,
			ip:  *out.PublicIp,
		})
	}

	return resources, nil
}

func (e *EC2Address) Details() string {
	return e.obj.String()
}

func (e *EC2Address) ID() string {
	return e.ip
}
