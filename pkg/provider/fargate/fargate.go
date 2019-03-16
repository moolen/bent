package fargate

import (
	"fmt"
	"strconv"
	"strings"

	log "github.com/sirupsen/logrus"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/ecs"

	"github.com/moolen/bent/pkg/provider"
)

// Provider implements provider.EndpointProvider
type Provider struct {
	Session *session.Session
	Client  *ecs.ECS
}

// NewProvider returns a new provider
func NewProvider() (*Provider, error) {
	session, err := newSession()
	if err != nil {
		return nil, err
	}
	client := ecs.New(session)
	return &Provider{
		Session: session,
		Client:  client,
	}, nil
}

// GetClusters returns a list of services running in FARGATE cluster
// assumptions:
//   - we DON'T care about the FARGATE service concept
//   - a container within a task can expose _multiple_ services
func (p Provider) GetClusters() (map[string][]provider.Cluster, error) {
	localClusters := make(map[string][]provider.Cluster)
	clusters, err := p.listClusters()
	if err != nil {
		return nil, err
	}

	for _, cluster := range clusters {
		serviceTasks, err := p.listTasks(*cluster.ClusterArn)
		if err != nil {
			return nil, err
		}
		serviceTaskDefs, err := p.getTaskDefinitions(keys(serviceTasks))
		if err != nil {
			return nil, err
		}
		log.Debugf("cluster %s has tasks: %#v", *cluster.ClusterName, serviceTasks)

		// get all endpoints per task
		for _, tasks := range serviceTasks {
			for _, task := range tasks {
				taskdef := serviceTaskDefs[*task.TaskDefinitionArn]
				// find related endpoints
				taskEndpoints, err := p.findEndpoints(task, taskdef)
				if err != nil {
					log.Warnf("error finding endpoints for task %s: %s", *task.TaskArn, err)
					continue
				}
				nodeID, err := TaskArnToNodeID(*task.TaskArn)
				if err != nil {
					log.Warnf("error parsing TaskArn %s: %s", *task.TaskArn, err)
					continue
				}

				// defaults: every task may launch a sidecar
				localClusters[nodeID] = []provider.Cluster{}

				for name, endpoints := range taskEndpoints {
					localClusters[nodeID] = append(localClusters[nodeID], provider.Cluster{
						Name:      name,
						Endpoints: endpoints,
					})
				}
			}
		}
	}
	return localClusters, nil
}

// TaskArnToNodeID transforms a TaskArn to a node id
func TaskArnToNodeID(arn string) (string, error) {
	parts := strings.Split(arn, "task/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid task arn: %s", arn)
	}
	return parts[1], nil
}

func (p Provider) findEndpoints(task *ecs.Task, taskdef *ecs.TaskDefinition) (map[string][]provider.Endpoint, error) {
	services := make(map[string][]provider.Endpoint)
	log.Infof("finding endpoints for task %#v", *task)
	taskTargets, err := findTaskTargets(taskdef)
	if err != nil {
		return nil, fmt.Errorf("error parsing tags of task %s: %s", *task.TaskDefinitionArn, err)
	}
	log.Infof("task taskTargets: %#v", taskTargets)
	for _, container := range task.Containers {
		if taskTargets[*container.Name] != nil {
			for _, target := range taskTargets[*container.Name] {
				for _, nic := range container.NetworkInterfaces {
					services[target.ClusterName] = append(services[target.ClusterName], provider.Endpoint{
						Address:     *nic.PrivateIpv4Address,
						Annotations: target.Annotations,
						Port:        target.Port,
					})
				}
			}
		}
	}

	return services, nil
}

type taskTarget struct {
	ClusterName string
	Annotations map[string]string
	Port        uint32
}

// should start with envoy.service-{svc}
func findTaskTargets(task *ecs.TaskDefinition) (map[string][]taskTarget, error) {
	targets := make(map[string][]taskTarget)
	for _, container := range task.ContainerDefinitions {
		for label, value := range container.DockerLabels {
			// FIXME: properly validate labels
			//        envoy.service.foo.bar.annotations.baz.bang = foo:123
			//        -> this would create a service foo.bar.annotations.baz.bang
			//           that points to container "foo" at port "123"
			clusterName := strings.TrimPrefix(label, "envoy.service.")
			if label == clusterName {
				continue
			}
			list := strings.Split(*value, ":")
			if len(list) < 2 {
				continue
			}
			port, err := strconv.Atoi(list[1])
			if err != nil {
				continue
			}
			targets[list[0]] = append(targets[list[0]], taskTarget{
				ClusterName: clusterName,
				Annotations: stripKeyPrefix(fmt.Sprintf("%s.annotations.", label), container.DockerLabels),
				Port:        uint32(port),
			})
		}
	}
	return targets, nil
}

// stripKeyPrefix removes the prefixes from all keys in the labels map
// if a key don't have a prefix they're being ejected
func stripKeyPrefix(prefix string, labels map[string]*string) map[string]string {
	annotations := make(map[string]string)

	for key, val := range labels {
		annotation := strings.TrimPrefix(key, prefix)
		if annotation == key {
			continue
		}
		annotations[annotation] = *val
	}

	return annotations
}
