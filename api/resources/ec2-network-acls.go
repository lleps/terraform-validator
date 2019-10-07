package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2NetworkACL struct {
	svc       *ec2.EC2
	id        *string
	isDefault *bool
	details   string
}

func init() {
	register("EC2NetworkACL", ListEC2NetworkACLs)
}

func ListEC2NetworkACLs(sess *session.Session) ([]Resource, error) {
	svc := ec2.New(sess)

	resp, err := svc.DescribeNetworkAcls(nil)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, out := range resp.NetworkAcls {
		resources = append(resources, &EC2NetworkACL{
			svc:       svc,
			id:        out.NetworkAclId,
			isDefault: out.IsDefault,
			details:   out.String(),
		})
	}

	return resources, nil
}

func (e *EC2NetworkACL) ID() string {
	return *e.id
}

func (e *EC2NetworkACL) Details() string {
	return e.details
}
