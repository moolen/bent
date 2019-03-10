package fargate

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

const (
	describeClustersWindowSize = 100
)

func (p Provider) listClusters() (map[string]*ecs.Cluster, error) {
	result := map[string]*ecs.Cluster{}
	clusters, err := p.listClusterArns()
	if err != nil {
		return nil, err
	}

	err = sliceWalk(describeClustersWindowSize, clusters, func(ids []*string) error {
		arg := &ecs.DescribeClustersInput{Clusters: ids}

		out, err := p.Client.DescribeClusters(arg)
		if err != nil {
			return fmt.Errorf("failed to describe clusters: %s", err.Error())
		}

		for _, f := range out.Failures {
			log.Warnf("%v: %v", *f.Arn, *f.Reason)
		}

		for _, s := range out.Clusters {
			result[*s.ClusterArn] = s
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (p Provider) listClusterArns() ([]*string, error) {
	arg := &ecs.ListClustersInput{}
	ecsClusters := []*string{}

	for {
		out, err := p.Client.ListClusters(arg)
		if err != nil {
			return nil, fmt.Errorf("could not retrieve ECS clusters: %s", err.Error())
		}

		ecsClusters = append(ecsClusters, out.ClusterArns...)
		arg.NextToken = out.NextToken
		if arg.NextToken == nil {
			break
		}
	}

	return ecsClusters, nil
}
