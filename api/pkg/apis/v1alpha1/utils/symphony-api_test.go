/*
 * Copyright (c) Microsoft Corporation.
 * Licensed under the MIT license.
 * SPDX-License-Identifier: MIT
 */

package utils

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/eclipse-symphony/symphony/api/pkg/apis/v1alpha1/model"
	"github.com/stretchr/testify/require"
)

const (
	baseUrl  = "http://localhost:8090/v1alpha2/"
	user     = "admin"
	password = ""
)

func TestGetInstancesWhenSomeInstances(t *testing.T) {
	testSymphonyApi := os.Getenv("TEST_SYMPHONY_API")
	if testSymphonyApi != "yes" {
		t.Skip("Skipping becasue TEST_SYMPHONY_API is missing or not set to 'yes'")
	}

	solutionName := "solution1"
	solution1JsonObj := map[string]interface{}{
		"name": "e4i-assets",
		"type": "helm.v3",
		"properties": map[string]interface{}{
			"chart": map[string]interface{}{
				"repo":    "e4ipreview.azurecr.io/helm/az-e4i-demo-assets",
				"version": "0.4.0",
			},
		},
	}

	solution1, err := json.Marshal(solution1JsonObj)
	if err != nil {
		panic(err)
	}

	err = UpsertSolution(context.Background(), baseUrl, solutionName, user, password, solution1, "default")
	require.NoError(t, err)

	targetName := "target1"
	target1JsonObj := map[string]interface{}{
		"id": "self",
		"spec": map[string]interface{}{
			"displayName": "int-virtual-02",
			"scope":       "alice-springs",
			"components": []interface{}{
				map[string]interface{}{
					"name": "e4k",
					"type": "helm.v3",
					"properties": map[string]interface{}{
						"chart": map[string]interface{}{
							"repo":    "e4kpreview.azurecr.io/helm/az-e4k",
							"version": "0.3.0",
						},
					},
				},
				map[string]interface{}{
					"name": "e4i",
					"type": "helm.v3",
					"properties": map[string]interface{}{
						"chart": map[string]interface{}{
							"repo":    "e4ipreview.azurecr.io/helm/az-e4i",
							"version": "0.4.0",
						},
						"values": map[string]interface{}{
							"mqttBroker": map[string]interface{}{
								"authenticationMethod": "serviceAccountToken",
								"name":                 "azedge-dmqtt-frontend",
								"namespace":            "alice-springs",
							},
							"opcPlcSimulation": map[string]interface{}{
								"deploy": "true",
							},
							"openTelemetry": map[string]interface{}{
								"enabled":  "true",
								"endpoint": "http://otel-collector.alice-springs.svc.cluster.local:4317",
								"protocol": "grpc",
							},
						},
					},
				},
				map[string]interface{}{
					"name": "bluefin",
					"type": "helm.v3",
					"properties": map[string]interface{}{
						"chart": map[string]interface{}{
							"repo":    "azbluefin.azurecr.io/helmcharts/bluefin-arc-extension/bluefin-arc-extension",
							"version": "0.1.2",
						},
					},
				},
			},
			"topologies": []interface{}{
				map[string]interface{}{
					"bindings": []interface{}{
						map[string]interface{}{
							"role":     "instance",
							"provider": "providers.target.k8s",
							"config": map[string]interface{}{
								"inCluster": "true",
							},
						},
						map[string]interface{}{
							"role":     "helm.v3",
							"provider": "providers.target.helm",
							"config": map[string]interface{}{
								"inCluster": "true",
							},
						},
						map[string]interface{}{
							"role":     "yaml.k8s",
							"provider": "providers.target.kubectl",
							"config": map[string]interface{}{
								"inCluster": "true",
							},
						},
					},
				},
			},
		},
	}
	target1, err := json.Marshal(target1JsonObj)
	require.NoError(t, err)

	err = CreateTarget(context.Background(), baseUrl, targetName, user, password, target1, "default")
	require.NoError(t, err)

	instanceName := "instance1"
	instance1JsonObj := map[string]interface{}{
		"scope":    "default",
		"solution": solutionName,
		"target": map[string]interface{}{
			"name": targetName,
		},
	}

	instance1, err := json.Marshal(instance1JsonObj)
	if err != nil {
		panic(err)
	}

	err = CreateInstance(context.Background(), baseUrl, instanceName, user, password, instance1, "default")
	require.NoError(t, err)

	// ensure instance gets created properly
	time.Sleep(time.Second)

	instancesRes, err := GetInstances(context.Background(), baseUrl, user, password, "default")
	require.NoError(t, err)

	require.Equal(t, 1, len(instancesRes))
	require.Equal(t, instanceName, instancesRes[0].Spec.DisplayName)
	require.Equal(t, solutionName, instancesRes[0].Spec.Solution)
	require.Equal(t, targetName, instancesRes[0].Spec.Target.Name)
	require.Equal(t, "1", instancesRes[0].Status["targets"])
	require.Equal(t, "OK", instancesRes[0].Status["status"])

	instanceRes, err := GetInstance(context.Background(), baseUrl, instanceName, user, password, "default")
	require.NoError(t, err)

	require.Equal(t, instanceName, instanceRes.Spec.DisplayName)
	require.Equal(t, solutionName, instanceRes.Spec.Solution)
	require.Equal(t, targetName, instanceRes.Spec.Target.Name)
	require.Equal(t, "1", instanceRes.Status["targets"])
	require.Equal(t, "OK", instanceRes.Status["status"])

	err = DeleteTarget(context.Background(), baseUrl, targetName, user, password, "default")
	require.NoError(t, err)

	err = DeleteSolution(context.Background(), baseUrl, solutionName, user, password, "default")
	require.NoError(t, err)

	err = DeleteInstance(context.Background(), baseUrl, instanceName, user, password, "default")
	require.NoError(t, err)
}

func TestGetSolutionsWhenSomeSolution(t *testing.T) {
	testSymphonyApi := os.Getenv("TEST_SYMPHONY_API")
	if testSymphonyApi != "yes" {
		t.Skip("Skipping becasue TEST_SYMPHONY_API is missing or not set to 'yes'")
	}

	solutionName := "solution1"
	solution1JsonObj := map[string]interface{}{
		"name": "e4i-assets",
		"type": "helm.v3",
		"properties": map[string]interface{}{
			"chart": map[string]interface{}{
				"repo":    "e4ipreview.azurecr.io/helm/az-e4i-demo-assets",
				"version": "0.4.0",
			},
		},
	}

	solution1, err := json.Marshal(solution1JsonObj)
	if err != nil {
		panic(err)
	}

	err = UpsertSolution(context.Background(), baseUrl, solutionName, user, password, solution1, "default")
	require.NoError(t, err)

	solutionsRes, err := GetSolutions(context.Background(), baseUrl, user, password, "default")
	require.NoError(t, err)

	require.Equal(t, 1, len(solutionsRes))
	require.Equal(t, solutionName, solutionsRes[0].Spec.DisplayName)

	solutionRes, err := GetSolution(context.Background(), baseUrl, solutionName, user, password, "default")
	require.NoError(t, err)

	require.Equal(t, solutionName, solutionRes.Spec.DisplayName)

	err = DeleteSolution(context.Background(), baseUrl, solutionName, user, password, "default")
	require.NoError(t, err)
}

func TestGetTargetsWithSomeTargets(t *testing.T) {
	testSymphonyApi := os.Getenv("TEST_SYMPHONY_API")
	if testSymphonyApi != "yes" {
		t.Skip("Skipping becasue TEST_SYMPHONY_API is missing or not set to 'yes'")
	}

	targetName := "target1"
	target1JsonObj := map[string]interface{}{
		"id": "self",
		"spec": map[string]interface{}{
			"displayName": "int-virtual-02",
			"scope":       "alice-springs",
			"components": []interface{}{
				map[string]interface{}{
					"name": "e4k",
					"type": "helm.v3",
					"properties": map[string]interface{}{
						"chart": map[string]interface{}{
							"repo":    "e4kpreview.azurecr.io/helm/az-e4k",
							"version": "0.3.0",
						},
					},
				},
				map[string]interface{}{
					"name": "e4i",
					"type": "helm.v3",
					"properties": map[string]interface{}{
						"chart": map[string]interface{}{
							"repo":    "e4ipreview.azurecr.io/helm/az-e4i",
							"version": "0.4.0",
						},
						"values": map[string]interface{}{
							"mqttBroker": map[string]interface{}{
								"authenticationMethod": "serviceAccountToken",
								"name":                 "azedge-dmqtt-frontend",
								"namespace":            "alice-springs",
							},
							"opcPlcSimulation": map[string]interface{}{
								"deploy": "true",
							},
							"openTelemetry": map[string]interface{}{
								"enabled":  "true",
								"endpoint": "http://otel-collector.alice-springs.svc.cluster.local:4317",
								"protocol": "grpc",
							},
						},
					},
				},
				map[string]interface{}{
					"name": "bluefin",
					"type": "helm.v3",
					"properties": map[string]interface{}{
						"chart": map[string]interface{}{
							"repo":    "azbluefin.azurecr.io/helmcharts/bluefin-arc-extension/bluefin-arc-extension",
							"version": "0.1.2",
						},
					},
				},
			},
			"topologies": []interface{}{
				map[string]interface{}{
					"bindings": []interface{}{
						map[string]interface{}{
							"role":     "instance",
							"provider": "providers.target.k8s",
							"config": map[string]interface{}{
								"inCluster": "true",
							},
						},
						map[string]interface{}{
							"role":     "helm.v3",
							"provider": "providers.target.helm",
							"config": map[string]interface{}{
								"inCluster": "true",
							},
						},
						map[string]interface{}{
							"role":     "yaml.k8s",
							"provider": "providers.target.kubectl",
							"config": map[string]interface{}{
								"inCluster": "true",
							},
						},
					},
				},
			},
		},
	}
	target1, err := json.Marshal(target1JsonObj)
	require.NoError(t, err)

	err = CreateTarget(context.Background(), baseUrl, targetName, user, password, target1, "default")
	require.NoError(t, err)

	// Ensure target gets created properly
	time.Sleep(time.Second)

	targetsRes, err := GetTargets(context.Background(), baseUrl, user, password, "default")
	require.NoError(t, err)

	require.Equal(t, 1, len(targetsRes))
	require.Equal(t, targetName, targetsRes[0].Spec.DisplayName)
	require.Equal(t, "default", targetsRes[0].Spec.Scope)
	require.Equal(t, "1", targetsRes[0].Status["targets"])
	require.Equal(t, "OK", targetsRes[0].Status["status"])

	targetRes, err := GetTarget(context.Background(), baseUrl, targetName, user, password, "default")
	require.NoError(t, err)

	require.Equal(t, targetName, targetRes.Spec.DisplayName)
	require.Equal(t, "default", targetRes.Spec.Scope)
	require.Equal(t, "1", targetRes.Status["targets"])
	require.Equal(t, "OK", targetRes.Status["status"])

	err = DeleteTarget(context.Background(), baseUrl, targetName, user, password, "default")
	require.NoError(t, err)
}

func TestMatchTargetsWithTargetName(t *testing.T) {
	res := MatchTargets(model.InstanceState{
		Id: "someId",
		Spec: &model.InstanceSpec{
			Target: model.TargetSelector{
				Name: "someTargetName",
			},
		},
		Status: map[string]string{},
	}, []model.TargetState{{
		Id: "someTargetName",
		Metadata: map[string]string{
			"key": "value",
		},
		Spec: &model.TargetSpec{},
	}})

	require.Equal(t, []model.TargetState{{
		Id: "someTargetName",
		Metadata: map[string]string{
			"key": "value",
		},
		Spec: &model.TargetSpec{},
	}}, res)
}

func TestMatchTargetsWithUnmatchedName(t *testing.T) {
	res := MatchTargets(model.InstanceState{
		Id: "someId",
		Spec: &model.InstanceSpec{
			Target: model.TargetSelector{
				Name: "someTargetName",
			},
		},
		Status: map[string]string{},
	}, []model.TargetState{{
		Id:   "someDifferentTargetName",
		Spec: &model.TargetSpec{},
	}})

	require.Equal(t, []model.TargetState{}, res)
}

func TestMatchTargetsWithSelectors(t *testing.T) {
	res := MatchTargets(model.InstanceState{
		Id: "someId",
		Spec: &model.InstanceSpec{
			Target: model.TargetSelector{
				Name: "someTargetName",
				Selector: map[string]string{
					"OS": "windows",
				},
			},
		},
		Status: map[string]string{},
	}, []model.TargetState{{
		Id: "someDifferentTargetName",
		Spec: &model.TargetSpec{
			DisplayName: "someDisplayName",
			Properties: map[string]string{
				"OS": "windows",
			},
		},
	}})

	require.Equal(t, []model.TargetState{{
		Id: "someDifferentTargetName",
		Spec: &model.TargetSpec{
			DisplayName: "someDisplayName",
			Properties: map[string]string{
				"OS": "windows",
			},
		},
	}}, res)
}

func TestMatchTargetsWithUnmatchedSelectors(t *testing.T) {
	res := MatchTargets(model.InstanceState{
		Id: "someId",
		Spec: &model.InstanceSpec{
			Target: model.TargetSelector{
				Name: "someTargetName",
				Selector: map[string]string{
					"OS": "windows",
				},
			},
		},
		Status: map[string]string{},
	}, []model.TargetState{{
		Id: "someDifferentTargetName",
		Spec: &model.TargetSpec{
			Properties: map[string]string{
				"OS": "linux",
			},
		},
	}})

	require.Equal(t, []model.TargetState{}, res)

	res = MatchTargets(model.InstanceState{
		Id: "someId",
		Spec: &model.InstanceSpec{
			Target: model.TargetSelector{
				Name: "someTargetName",
				Selector: map[string]string{
					"OS": "windows",
				},
			},
		},
		Status: map[string]string{},
	}, []model.TargetState{{
		Id: "someDifferentTargetName",
		Spec: &model.TargetSpec{
			Properties: map[string]string{
				"company": "linux",
			},
		},
	}})

	require.Equal(t, []model.TargetState{}, res)
}

func TestCreateSymphonyDeploymentFromTarget(t *testing.T) {
	res, err := CreateSymphonyDeploymentFromTarget(model.TargetState{
		Id: "someTargetName",
		Spec: &model.TargetSpec{
			DisplayName: "someDisplayName",
			Scope:       "targetScope",
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Components: []model.ComponentSpec{
				{
					Name: "componentName1",
					Type: "componentType1",
					Metadata: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
				{
					Name: "componentName2",
					Type: "componentType2",
				},
			},
			Properties: map[string]string{
				"OS": "windows",
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, model.DeploymentSpec{
		SolutionName: "target-runtime-someTargetName",
		Solution: model.SolutionSpec{
			DisplayName: "target-runtime-someTargetName",
			Scope:       "targetScope",
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
			},
			Components: []model.ComponentSpec{
				{
					Name: "componentName1",
					Type: "componentType1",
					Metadata: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
				{
					Name: "componentName2",
					Type: "componentType2",
				},
			},
		},
		Instance: model.InstanceSpec{
			Name:        "target-runtime-someTargetName",
			DisplayName: "target-runtime-someTargetName",
			Scope:       "targetScope",
			Solution:    "target-runtime-someTargetName",
			Target: model.TargetSelector{
				Name: "someTargetName",
			},
		},
		Targets: map[string]model.TargetSpec{
			"someTargetName": {
				DisplayName: "someDisplayName",
				Scope:       "targetScope",
				Metadata: map[string]string{
					"key1": "value1",
					"key2": "value2",
				},
				Properties: map[string]string{
					"OS": "windows",
				},
				Components: []model.ComponentSpec{
					{
						Name: "componentName1",
						Type: "componentType1",
						Metadata: map[string]string{
							"key1": "value1",
							"key2": "value2",
						},
					},
					{
						Name: "componentName2",
						Type: "componentType2",
					},
				},
				ForceRedeploy: false,
			},
		},
		Assignments: map[string]string{
			"someTargetName": "{componentName1}{componentName2}",
		},
	}, res)
}

func TestCreateSymphonyDeployment(t *testing.T) {
	res, err := CreateSymphonyDeployment(model.InstanceState{
		Id: "someOtherId",
		Spec: &model.InstanceSpec{
			Target: model.TargetSelector{
				Name: "someTargetName",
				Selector: map[string]string{
					"OS": "windows",
				},
			},
			Scope: "instanceScope",
		},
		Status: map[string]string{},
	}, model.SolutionState{
		Id: "someOtherId",
		Spec: &model.SolutionSpec{
			DisplayName: "someDisplayName",
			Scope:       "solutionsScope",
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Components: []model.ComponentSpec{
				{
					Name: "componentName1",
					Type: "componentType1",
				},
				{
					Name: "componentName2",
					Type: "componentType2",
					Metadata: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
		},
	}, []model.TargetState{
		{
			Id: "someTargetName1",
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Spec: &model.TargetSpec{
				Properties: map[string]string{
					"company": "microsoft",
				},
				Scope: "targetScope",
			},
		},
	}, []model.DeviceState{
		{
			Id: "someTargetName2",
			Spec: &model.DeviceSpec{
				DisplayName: "someDeviceDisplayName",
				Properties: map[string]string{
					"company": "microsoft",
				},
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, model.DeploymentSpec{
		SolutionName: "someOtherId",
		Solution: model.SolutionSpec{
			DisplayName: "someDisplayName",
			Scope:       "solutionsScope",
			Metadata: map[string]string{
				"key1": "value1",
				"key2": "value2",
				"key3": "value3",
			},
			Components: []model.ComponentSpec{
				{
					Name: "componentName1",
					Type: "componentType1",
				},
				{
					Name: "componentName2",
					Type: "componentType2",
					Metadata: map[string]string{
						"key1": "value1",
						"key2": "value2",
					},
				},
			},
		},
		Instance: model.InstanceSpec{
			Name:        "someOtherId",
			DisplayName: "",
			Scope:       "instanceScope",
			Solution:    "",
			Target: model.TargetSelector{
				Name: "someTargetName",
				Selector: map[string]string{
					"OS": "windows",
				},
			},
		},
		Targets: map[string]model.TargetSpec{
			"someTargetName1": {
				Scope: "targetScope",
				Properties: map[string]string{
					"company": "microsoft",
				},
				ForceRedeploy: false,
			},
		},
		Assignments: map[string]string{
			"someTargetName1": "{componentName1}{componentName2}",
		},
	}, res)
}

func TestAssignComponentsToTargetsWithMixedConstraints(t *testing.T) {
	res, err := AssignComponentsToTargets([]model.ComponentSpec{
		{
			Name:        "componentName1",
			Constraints: "${{$equal($property(OS),windows)}}",
		},
		{
			Name:        "componentName2",
			Constraints: "${{$equal($property(OS),linux)}}",
		},
		{
			Name:        "componentName3",
			Constraints: "${{$equal($property(OS),unix)}}",
		},
	}, map[string]model.TargetSpec{
		"target1": {
			Properties: map[string]string{
				"OS": "windows",
			},
		},
		"target2": {
			Properties: map[string]string{
				"OS": "linux",
			},
		},
		"target3": {
			Properties: map[string]string{
				"OS": "unix",
			},
		},
	})
	require.NoError(t, err)

	require.Equal(t, map[string]string{
		"target1": "{componentName1}",
		"target2": "{componentName2}",
		"target3": "{componentName3}",
	}, res)
}
