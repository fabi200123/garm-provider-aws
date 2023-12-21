// Copyright 2023 Cloudbase Solutions SRL
//
//    Licensed under the Apache License, Version 2.0 (the "License"); you may
//    not use this file except in compliance with the License. You may obtain
//    a copy of the License at
//
//         http://www.apache.org/licenses/LICENSE-2.0
//
//    Unless required by applicable law or agreed to in writing, software
//    distributed under the License is distributed on an "AS IS" BASIS, WITHOUT
//    WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the
//    License for the specific language governing permissions and limitations
//    under the License.

package provider

import (
	"context"
	"fmt"

	"github.com/cloudbase/garm-provider-aws/config"
	"github.com/cloudbase/garm-provider-aws/internal/client"
	"github.com/cloudbase/garm-provider-aws/internal/spec"
	"github.com/cloudbase/garm-provider-common/execution"
	"github.com/cloudbase/garm-provider-common/params"
)

var _ execution.ExternalProvider = &AwsProvider{}

func NewAwsProvider(configPath, controllerID string) (execution.ExternalProvider, error) {
	conf, err := config.NewConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}
	awsCli, err := client.NewAwsCli(conf)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS CLI: %w", err)
	}

	return &AwsProvider{
		controllerID: controllerID,
		awsCli:       awsCli,
	}, nil
}

type AwsProvider struct {
	cfg          *config.Config
	controllerID string
	awsCli       *client.AwsCli
}

func (a *AwsProvider) CreateInstance(ctx context.Context, bootstrapParams params.BootstrapInstance) (params.ProviderInstance, error) {
	if bootstrapParams.OSArch != params.Amd64 {
		return params.ProviderInstance{}, fmt.Errorf("unsupported architecture: %s", bootstrapParams.OSArch)
	}

	spec, err := spec.GetRunnerSpecFromBootstrapParams(*a.cfg, bootstrapParams, a.controllerID)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to get runner spec: %w", err)
	}

	subnetID, err := a.awsCli.CreateSubnet(ctx, spec.VpcID, "10.10.0.0/24", spec.Region)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to create subnet: %w", err)
	}

	groupID, err := a.awsCli.CreateSecurityGroup(ctx, spec.VpcID, spec)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to create security group: %w", err)
	}

	instanceID, err := a.awsCli.CreateRunningInstance(ctx, spec, subnetID, groupID)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to create instance: %w", err)
	}

	instance := params.ProviderInstance{
		ProviderID: instanceID,
		Name:       spec.BootstrapParams.Name,
		Status:     "running",
	}

	return instance, nil

}

func (a *AwsProvider) DeleteInstance(ctx context.Context, instance string) error {
	// Clear the security group
	if err := a.awsCli.DeleteSecurityGroup(ctx, a.cfg.VpcID); err != nil {
		return fmt.Errorf("failed to delete security group: %w", err)
	}

	// Terminate the instance
	if err := a.awsCli.TerminateInstance(ctx, instance); err != nil {
		return fmt.Errorf("failed to delete instance: %w", err)
	}
	return nil
}

// TODO: Implement this
func (a *AwsProvider) GetInstance(ctx context.Context, instance string) (params.ProviderInstance, error) {
	_, err := a.awsCli.GetInstance(ctx, instance)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to get VM details: %w", err)
	}
	//TODO: write function to convert aws.Instance to params.ProviderInstance
	details, err := params.ProviderInstance{}, nil
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to convert VM details: %w", err)
	}
	return details, nil
}

// TODO: Implement this
func (a *AwsProvider) ListInstances(ctx context.Context, poolID string) ([]params.ProviderInstance, error) {
	instances, err := a.awsCli.ListDescribedInstances(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	if instances == nil {
		return []params.ProviderInstance{}, nil
	}

	resp := make([]params.ProviderInstance, len(instances))
	for idx := range instances {
		//TODO: write function to convert aws.Instance to params.ProviderInstance
		resp[idx] = params.ProviderInstance{}
	}

	return resp, nil
}

func (a *AwsProvider) RemoveAllInstances(ctx context.Context) error {
	return nil
}

func (a *AwsProvider) Stop(ctx context.Context, instance string, force bool) error {
	return a.awsCli.StopInstance(ctx, instance)

}

func (a *AwsProvider) Start(ctx context.Context, instance string) error {
	return a.awsCli.StartInstance(ctx, instance)
}
