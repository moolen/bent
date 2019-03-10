package fargate

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

const (
	describeServicesWindowSize = 10
)

func (p Provider) listServices(cluster string) (map[string]*ecs.Service, error) {
	result := map[string]*ecs.Service{}
	services, err := p.listServiceArns(cluster)
	if err != nil {
		return nil, err
	}

	err = sliceWalk(describeServicesWindowSize, services, func(ids []*string) error {
		arg := &ecs.DescribeServicesInput{Cluster: &cluster, Services: ids}

		out, err := p.Client.DescribeServices(arg)
		if err != nil {
			return fmt.Errorf("failed to describe %s services: %s", cluster, err.Error())
		}

		for _, f := range out.Failures {
			log.Warnf("%v: %v", *f.Arn, *f.Reason)
		}

		for _, s := range out.Services {
			result[*s.ServiceArn] = s
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p Provider) listServiceArns(cluster string) ([]*string, error) {
	services := []*string{}
	arg := &ecs.ListServicesInput{Cluster: &cluster}

	hasNext := true
	for hasNext {
		out, err := p.Client.ListServices(arg)
		if err != nil {
			return nil, fmt.Errorf(
				"could not list services for cluster %s: %s", cluster, err.Error())
		}
		services = append(services, out.ServiceArns...)
		arg.NextToken = out.NextToken
		hasNext = arg.NextToken != nil
	}

	return services, nil
}
