package resources

import (
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/elb"
	"strconv"
	"strings"
)

type ELBLoadBalancer struct {
	svc  *elb.ELB
	name *string
	tags []*elb.Tag
}

func init() {
	register("ELB", ListELBLoadBalancers)
}

func ListELBLoadBalancers(sess *session.Session) ([]Resource, error) {
	resources := make([]Resource, 0)

	svc := elb.New(sess)

	elbResp, err := svc.DescribeLoadBalancers(nil)
	if err != nil {
		return nil, err
	}

	for _, elbLoadBalancer := range elbResp.LoadBalancerDescriptions {
		// Tags for ELBs need to be fetched separately
		tagResp, err := svc.DescribeTags(&elb.DescribeTagsInput{
			LoadBalancerNames: []*string{elbLoadBalancer.LoadBalancerName},
		})

		if err != nil {
			return nil, err
		}

		for _, elbTagInfo := range tagResp.TagDescriptions {
			resources = append(resources, &ELBLoadBalancer{
				svc:  svc,
				name: elbTagInfo.LoadBalancerName,
				tags: elbTagInfo.Tags,
			})
		}
	}

	return resources, nil
}

func (e *ELBLoadBalancer) Remove() error {
	params := &elb.DeleteLoadBalancerInput{
		LoadBalancerName: e.name,
	}

	_, err := e.svc.DeleteLoadBalancer(params)
	if err != nil {
		return err
	}

	return nil
}

func (e *ELBLoadBalancer) Details() string {
	result := strings.Builder{}
	result.WriteString("Tags:\n")
	for i, t := range e.tags {
		result.WriteString("#" + strconv.Itoa(i+1) + ": " + *t.Key + " => " + *t.Value)
		result.WriteRune('\n')
	}
	return result.String()
}

func (e *ELBLoadBalancer) ID() string {
	return *e.name
}
