package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ec2"
)

type EC2Instance struct {
	svc      *ec2.EC2
	instance *ec2.Instance
}

func (instance EC2Instance) ID() string {
	return *instance.instance.InstanceId
}

func (instance EC2Instance) Details() string {
	return instance.instance.String()
}

func init() {
	register("EC2Instance", ListEC2Instances)
}

func ListEC2Instances(sess *session.Session) ([]Resource, error) {
	svc := ec2.New(sess)

	params := &ec2.DescribeInstancesInput{}
	resp, err := svc.DescribeInstances(params)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, reservation := range resp.Reservations {
		for _, instance := range reservation.Instances {
			resources = append(resources, &EC2Instance{
				svc:      svc,
				instance: instance,
			})
		}
	}

	return resources, nil
}
