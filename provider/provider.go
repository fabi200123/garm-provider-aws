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
	"errors"
	"fmt"
	"strings"

	"github.com/cloudbase/garm-provider-aws/config"
	"github.com/cloudbase/garm-provider-aws/internal/client"
	"github.com/cloudbase/garm-provider-aws/internal/spec"
	garmErrors "github.com/cloudbase/garm-provider-common/errors"
	"github.com/cloudbase/garm-provider-common/execution"
	"github.com/cloudbase/garm-provider-common/params"
)

var _ execution.ExternalProvider = &AwsProvider{}

func NewAwsProvider(ctx context.Context, configPath, controllerID string) (execution.ExternalProvider, error) {
	conf, err := config.NewConfig(configPath)
	if err != nil {
		return nil, fmt.Errorf("error loading config: %w", err)
	}
	awsCli, err := client.NewAwsCli(ctx, conf)
	if err != nil {
		return nil, fmt.Errorf("failed to get AWS CLI: %w", err)
	}

	return &AwsProvider{
		cfg:          conf,
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

	spec, err := spec.GetRunnerSpecFromBootstrapParams(a.cfg, bootstrapParams, a.controllerID)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to get runner spec: %w", err)
	}

	instanceID, err := a.awsCli.CreateRunningInstance(ctx, spec)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to create instance: %w", err)
	}

	instance := params.ProviderInstance{
		ProviderID: instanceID,
		Name:       spec.BootstrapParams.Name,
		OSType:     spec.BootstrapParams.OSType,
		OSArch:     spec.BootstrapParams.OSArch,
		Status:     "running",
	}

	return instance, nil

}

func (a *AwsProvider) DeleteInstance(ctx context.Context, instance string) error {
	var inst string
	if strings.HasPrefix(instance, "i-") {
		inst = instance
	} else {
		tags := map[string]string{
			"GARM_CONTROLLER_ID": "",
			"Name":               instance,
		}

		tmp, err := a.awsCli.FindInstanceByTags(ctx, tags)
		if err != nil {
			if errors.Is(err, garmErrors.ErrNotFound) {
				return nil
			}
			return fmt.Errorf("failed to determine instance: %w", err)
		}
		inst = *tmp.InstanceId
	}

	if inst == "" {
		return nil
	}

	if err := a.awsCli.TerminateInstance(ctx, inst); err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

func (a *AwsProvider) GetInstance(ctx context.Context, instance string) (params.ProviderInstance, error) {
	awsInstance, err := a.awsCli.GetInstance(ctx, instance)
	if err != nil {
		return params.ProviderInstance{}, fmt.Errorf("failed to get VM details: %w", err)
	}
	if awsInstance == nil {
		return params.ProviderInstance{}, nil
	}

	providerInstance := params.ProviderInstance{
		ProviderID: *awsInstance.InstanceId,
		Name:       *awsInstance.Tags[0].Value,
		Status:     params.InstanceStatus(awsInstance.State.Name),
		OSType:     params.OSType(awsInstance.Platform),
		OSArch:     params.OSArch(awsInstance.Architecture),
		OSVersion:  *awsInstance.PlatformDetails,
	}
	return providerInstance, nil
}

func (a *AwsProvider) ListInstances(ctx context.Context, poolID string) ([]params.ProviderInstance, error) {
	awsInstances, err := a.awsCli.ListDescribedInstances(ctx, poolID)
	if err != nil {
		return nil, fmt.Errorf("failed to list instances: %w", err)
	}

	var providerInstances []params.ProviderInstance
	for _, awsInstance := range awsInstances {
		var name string
		if len(awsInstance.Tags) > 0 {
			name = *awsInstance.Tags[0].Value
		}

		pi := params.ProviderInstance{
			ProviderID: *awsInstance.InstanceId,
			Name:       name,
			Status:     params.InstanceStatus(awsInstance.State.Name),
			OSType:     params.OSType(awsInstance.Platform),
			OSArch:     params.OSArch(awsInstance.Architecture),
			OSVersion:  *awsInstance.PlatformDetails,
		}

		providerInstances = append(providerInstances, pi)
	}

	return providerInstances, nil
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
