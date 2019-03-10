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

const (
	maxDNSLength = 253
)

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

// GetServices returns a list of services running in FARGATE cluster
// assumptions:
//   - we DON'T care about the FARGATE service concept
//   - a container within a task can expose _multiple_ services
func (p Provider) GetServices() ([]provider.Service, map[string][]provider.Service, error) {
	nodes := make(map[string][]provider.Service)
	serviceMap := make(map[string][]provider.Endpoint)

	clusters, err := p.listClusters()
	if err != nil {
		return nil, nil, err
	}

	for _, cluster := range clusters {
		serviceTasks, err := p.listTasks(*cluster.ClusterArn)
		if err != nil {
			return nil, nil, err
		}
		serviceTaskDefs, err := p.getTaskDefinitions(keys(serviceTasks))
		log.Debugf("cluster %s has tasks: %#v", *cluster.ClusterName, serviceTasks)

		// get all endpoints per task
		for _, tasks := range serviceTasks {
			for _, task := range tasks {
				taskdef := serviceTaskDefs[*task.TaskDefinitionArn]
				// find related endpoints
				taskServices, err := p.findServices(task, taskdef)
				if err != nil {
					log.Warnf("error finding endpoints for task %s: %s", *task.TaskArn, err)
					continue
				}
				nodeID, err := TaskArnToNodeID(*task.TaskArn)
				if err != nil {
					log.Warnf("error parsing TaskArn %s: %s", *task.TaskArn, err)
					continue
				}

				for svc, taskEPs := range taskServices {
					serviceMap[svc] = append(serviceMap[svc], taskEPs...)

					nodes[nodeID] = append(nodes[nodeID], provider.Service{
						Name:        svc,
						Annotations: sumEndpointAnnotations(taskEPs),
						Endpoints:   taskEPs,
					})
				}
			}
		}
	}

	// transform map to array
	providerServices := []provider.Service{}

	for svc, eps := range serviceMap {

		providerServices = append(providerServices, provider.Service{
			Name:        svc,
			Annotations: sumEndpointAnnotations(eps),
			Endpoints:   eps,
		})
	}

	return providerServices, nodes, nil
}

// TaskArnToNodeID transforms a TaskArn to a node id
func TaskArnToNodeID(arn string) (string, error) {
	parts := strings.Split(arn, "task/")
	if len(parts) < 2 {
		return "", fmt.Errorf("invalid task arn: %s", arn)
	}
	return parts[1], nil
}

func (p Provider) findServices(task *ecs.Task, taskdef *ecs.TaskDefinition) (map[string][]provider.Endpoint, error) {
	services := make(map[string][]provider.Endpoint)
	log.Infof("finding endpoints for task %#v", *task)
	targets, err := findTarget(taskdef)
	if err != nil {
		return nil, fmt.Errorf("error parsing tags of task %s: %s", *task.TaskDefinitionArn, err)
	}
	log.Infof("task targets: %#v", targets)
	for _, container := range task.Containers {
		if targets[*container.Name] != nil {
			for _, svc := range targets[*container.Name] {
				for _, nic := range container.NetworkInterfaces {
					services[svc.ServiceName] = append(services[svc.ServiceName], provider.Endpoint{
						Address:     *nic.PrivateIpv4Address,
						Annotations: svc.Annotations,
						Port:        svc.Port,
					})
				}
			}
		}
	}

	return services, nil
}

type taskTarget struct {
	ServiceName string
	Annotations map[string]string
	Port        uint32
}

// tags should start with service-{svc}
func findTarget(task *ecs.TaskDefinition) (map[string][]taskTarget, error) {
	targets := make(map[string][]taskTarget)
	for _, container := range task.ContainerDefinitions {
		for label, value := range container.DockerLabels {
			// FIXME: properly validate labels
			//        envoy.service.foo.bar.annotations.baz.bang = foo:123
			//        -> this would create a service foo.bar.annotations.baz.bang
			//           that points to container "foo" at port "123"
			serviceName := strings.TrimPrefix(label, "envoy.service.")
			if label == serviceName {
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
				ServiceName: serviceName,
				Annotations: stripKeyPrefix(fmt.Sprintf("%s.annotations.", label), container.DockerLabels),
				Port:        uint32(port),
			})
		}

	}
	return targets, nil
}

func sumEndpointAnnotations(eps []provider.Endpoint) map[string]string {
	annotations := make(map[string]string)
	for _, ep := range eps {
		for k, v := range ep.Annotations {
			annotations[k] = v
		}
	}
	return annotations
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

func getTag(tags []*ecs.Tag, key, fallback string) string {
	for _, tag := range tags {
		if *tag.Key == key {
			return *tag.Value
		}
	}
	return fallback
}
