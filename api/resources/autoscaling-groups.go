package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/autoscaling"
)

func init() {
	register("AutoScalingGroup", ListAutoscalingGroups)
}

func ListAutoscalingGroups(s *session.Session) ([]Resource, error) {
	svc := autoscaling.New(s)

	params := &autoscaling.DescribeAutoScalingGroupsInput{}
	resp, err := svc.DescribeAutoScalingGroups(params)
	if err != nil {
		return nil, err
	}

	resources := make([]Resource, 0)
	for _, asg := range resp.AutoScalingGroups {
		resources = append(resources, &AutoScalingGroup{
			svc:  svc,
			asg:  asg,
			name: asg.AutoScalingGroupName,
		})
	}
	return resources, nil
}

type AutoScalingGroup struct {
	svc  *autoscaling.AutoScaling
	asg  *autoscaling.Group
	name *string
}

func (asg *AutoScalingGroup) Details() string {
	return asg.asg.String()
}

func (asg *AutoScalingGroup) ID() string {
	return *asg.name
}
