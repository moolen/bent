package fargate

import (
	"fmt"

	"github.com/aws/aws-sdk-go/service/ecs"
	log "github.com/sirupsen/logrus"
)

const (
	describeTasksWindowSize = 100
)

func (p Provider) listTasks(cluster string) (map[string][]*ecs.Task, error) {
	tasks := map[string][]*ecs.Task{}
	taskArns, err := p.listTaskArns(cluster)
	if err != nil {
		return nil, err
	}

	err = sliceWalk(describeTasksWindowSize, taskArns, func(ids []*string) error {
		arg := &ecs.DescribeTasksInput{Tasks: ids}
		if cluster != "" {
			arg.Cluster = &cluster
		}

		out, err := p.Client.DescribeTasks(arg)
		if err != nil {
			return fmt.Errorf("failed to load task instance data: %s", err.Error())
		}

		for _, f := range out.Failures {
			log.Warnf("%s: %s", *f.Arn, *f.Reason)
		}

		for _, t := range out.Tasks {
			if tasks[*t.TaskDefinitionArn] == nil {
				tasks[*t.TaskDefinitionArn] = []*ecs.Task{}
			}
			tasks[*t.TaskDefinitionArn] = append(tasks[*t.TaskDefinitionArn], t)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tasks, nil
}

func (p Provider) listTaskArns(cluster string) ([]*string, error) {
	arg := &ecs.ListTasksInput{Cluster: &cluster}
	tasks := []*string{}

	moreTasks := true
	for moreTasks {
		out, err := p.Client.ListTasks(arg)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, out.TaskArns...)
		arg.NextToken = out.NextToken
		moreTasks = arg.NextToken != nil
	}

	return tasks, nil
}

func (p Provider) getTaskDefinitions(arns []string) (map[string]*ecs.TaskDefinition, error) {
	taskDefs := make(map[string]*ecs.TaskDefinition)
	arg := &ecs.DescribeTaskDefinitionInput{}
	for _, arn := range arns {
		arg.TaskDefinition = &arn
		out, err := p.Client.DescribeTaskDefinition(arg)
		if err != nil {
			log.Warnf("faild to load task def %s: %s", arn, err)
			continue
		}
		taskDefs[arn] = out.TaskDefinition
	}

	return taskDefs, nil
}
