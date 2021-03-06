// Copyright 2017-2018 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"time"

	"github.com/moolen/bent/pkg/provider/fargate"
)

const (
	v2MetadataEndpoint     = "http://169.254.170.2/v2/metadata"
	maxRetries             = 4
	durationBetweenRetries = time.Second * 2
)

// TaskResponse defines the schema for the task response JSON object
type TaskResponse struct {
	Cluster       string
	TaskARN       string
	Family        string
	Revision      string
	DesiredStatus string `json:",omitempty"`
	KnownStatus   string
	Containers    []ContainerResponse `json:",omitempty"`
	Limits        LimitsResponse      `json:",omitempty"`
}

// ContainerResponse defines the schema for the container response
// JSON object
type ContainerResponse struct {
	ID            string `json:"DockerId"`
	Name          string
	DockerName    string
	Image         string
	ImageID       string
	Ports         []PortResponse    `json:",omitempty"`
	Labels        map[string]string `json:",omitempty"`
	DesiredStatus string
	KnownStatus   string
	ExitCode      *int `json:",omitempty"`
	Limits        LimitsResponse
	CreatedAt     *time.Time `json:",omitempty"`
	StartedAt     *time.Time `json:",omitempty"`
	FinishedAt    *time.Time `json:",omitempty"`
	Type          string
	Health        HealthStatus `json:"health,omitempty"`
	Networks      []Network    `json:",omitempty"`
}

// HealthStatus defines the schema for the container's health status
type HealthStatus struct {
	Status   string     `json:"status,omitempty"`
	Since    *time.Time `json:"statusSince,omitempty"`
	ExitCode int        `json:"exitCode,omitempty"`
	Output   string     `json:"output,omitempty"`
}

// LimitsResponse defines the schema for task/cpu limits response
// JSON object
type LimitsResponse struct {
	CPU    uint
	Memory uint
}

// PortResponse defines the schema for portmapping response JSON
// object
type PortResponse struct {
	ContainerPort uint16
	Protocol      string
	HostPort      uint16 `json:",omitempty"`
}

// Network is a struct that keeps track of metadata of a network interface
type Network struct {
	NetworkMode   string   `json:"NetworkMode,omitempty"`
	IPv4Addresses []string `json:"IPv4Addresses,omitempty"`
	IPv6Addresses []string `json:"IPv6Addresses,omitempty"`
}

func main() {
	client := &http.Client{
		Timeout: 5 * time.Second,
	}

	// Wait for the Health information to be ready
	time.Sleep(5 * time.Second)

	taskMetadata, err := taskMetadata(client)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Unable to get task metadata: %v", err)
		os.Exit(1)
	}

	nodeID, err := fargate.TaskArnToNodeID(taskMetadata.TaskARN)
	if err != nil {
		fmt.Fprintf(os.Stderr, "err")
		os.Exit(1)
	}

	// expose everything
	fmt.Printf("%s", nodeID)

	// node.Name = taskMetadata.Family
	// containerID := ""
	// for _, container := range taskMetadata.Containers {
	// 	if container.Type == "NORMAL" {
	// 		containerID = container.ID
	// 		break
	// 	}
	// 	containerMetadata, err := containerMetadata(client, containerID)
	// 	if err != nil {
	// 		fmt.Fprintf(os.Stderr, "Unable to get container metadata for '%s': %v", containerID, err)
	// 		os.Exit(1)
	// 	}
	// 	for _, network := range containerMetadata.Networks {
	// 		node.Addresses = append(node.Addresses, network.IPv4Addresses...)
	// 	}
	// }

	// fmt.Println(node.String())
}

func taskMetadata(client *http.Client) (*TaskResponse, error) {
	body, err := metadataResponse(client, v2MetadataEndpoint, "task metadata")
	if err != nil {
		return nil, err
	}

	var taskMetadata TaskResponse
	err = json.Unmarshal(body, &taskMetadata)
	if err != nil {
		return nil, fmt.Errorf("task metadata: unable to parse response body: %v", err)
	}

	return &taskMetadata, nil
}

func containerMetadata(client *http.Client, id string) (*ContainerResponse, error) {
	body, err := metadataResponse(client, v2MetadataEndpoint+"/"+id, "container metadata")
	if err != nil {
		return nil, err
	}

	var containerMetadata ContainerResponse
	err = json.Unmarshal(body, &containerMetadata)
	if err != nil {
		return nil, fmt.Errorf("container metadata: unable to parse response body: %v", err)
	}

	return &containerMetadata, nil
}

func metadataResponse(client *http.Client, endpoint string, respType string) ([]byte, error) {
	var resp []byte
	var err error
	for i := 0; i < maxRetries; i++ {
		resp, err = metadataResponseOnce(client, endpoint, respType)
		if err == nil {
			return resp, nil
		}
		fmt.Fprintf(os.Stderr, "Attempt [%d/%d]: unable to get metadata response for '%s' from '%s': %v",
			i, maxRetries, respType, endpoint, err)
		time.Sleep(durationBetweenRetries)
	}

	return nil, err
}

func metadataResponseOnce(client *http.Client, endpoint string, respType string) ([]byte, error) {
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("%s: unable to get response: %v", respType, err)
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: incorrect status code  %d", respType, resp.StatusCode)
	}
	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("task metadata: unable to read response body: %v", err)
	}

	return body, nil
}
