package resources

import (
	"fmt"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2SecurityGroup struct {
	svc     *ec2.EC2
	group   *ec2.SecurityGroup
	id      *string
	name    *string
	ingress []*ec2.IpPermission
	egress  []*ec2.IpPermission
}

func init() {
	register("EC2SecurityGroup", ListEC2SecurityGroups)
}

func ListEC2SecurityGroups(sess *session.Session) ([]Resource, error) {
	svc := ec2.New(sess)

	params := &ec2.DescribeSecurityGroupsInput{}
	resp, err := svc.DescribeSecurityGroups(params)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, group := range resp.SecurityGroups {
		resources = append(resources, &EC2SecurityGroup{
			svc:     svc,
			id:      group.GroupId,
			name:    group.GroupName,
			ingress: group.IpPermissions,
			egress:  group.IpPermissionsEgress,
			group:   group,
		})
	}

	return resources, nil
}

func (sg *EC2SecurityGroup) Filter() error {
	if *sg.name == "default" {
		return fmt.Errorf("cannot delete group 'default'")
	}

	return nil
}

func (sg *EC2SecurityGroup) Details() string {
	return sg.group.String()
}

func (sg *EC2SecurityGroup) ID() string {
	return *sg.id
}
