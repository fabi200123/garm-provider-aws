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

package client

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/cloudbase/garm-provider-aws/config"
	"github.com/cloudbase/garm-provider-aws/internal/spec"
)

func NewAwsCli(cfg *config.Config) (*AwsCli, error) {
	creds, err := cfg.Credentials.GetCredentials()
	if err != nil {
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	//TODO: Add credentials in format that ec2.Options accepts
	opts := ec2.Options{
		Region:      cfg.Region,
		Credentials: nil,
	}

	client := ec2.New(opts)

	awsCli := &AwsCli{
		cfg:    cfg,
		cred:   creds,
		client: *client,
		region: cfg.Region,
	}

	return awsCli, nil
}

type AwsCli struct {
	cfg  *config.Config
	cred aws.Credentials

	client ec2.Client
	region string
}

func (a *AwsCli) StartInstance(ctx context.Context, vmName string) error {
	_, err := a.client.StartInstances(ctx, &ec2.StartInstancesInput{
		InstanceIds: []string{vmName},
	})
	if err != nil {
		return fmt.Errorf("failed to start instance: %w", err)
	}

	return nil
}

func (a *AwsCli) StopInstance(ctx context.Context, vmName string) error {
	_, err := a.client.StopInstances(ctx, &ec2.StopInstancesInput{
		InstanceIds: []string{vmName},
		// Forces the instances to stop. The instances do not have an opportunity to flush
		// file system caches or file system metadata. If you use this option, you must
		// perform file system check and repair procedures. This option is not recommended
		// for Windows instances. Default: false
		//Force:       force,
	})
	if err != nil {
		return fmt.Errorf("failed to stop instance: %w", err)
	}

	return nil
}

// TODO: Find better way to get instance without the Instance struct
func (a *AwsCli) GetInstance(ctx context.Context, vmName string) (*types.Instance, error) {
	resp, err := a.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: []string{vmName},
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	if len(resp.Reservations) == 0 || len(resp.Reservations[0].Instances) == 0 {
		return nil, fmt.Errorf("instance not found")
	}

	return &resp.Reservations[0].Instances[0], nil
}

// You can stop, start, and terminate EBS-backed instances. You can only terminate instance store-backed instances. What happens to an instance differs if you stop it or terminate it. For example, when you stop an instance, the root device and any other devices attached to the instance persist. When you terminate an instance, any attached EBS volumes with the DeleteOnTermination block device mapping parameter set to true are automatically deleted.
func (a *AwsCli) TerminateInstance(ctx context.Context, vmName string) error {
	_, err := a.client.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: []string{vmName},
	})
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}

	return nil
}

// TODO: find better method to get all instances
func (a *AwsCli) ListDescribedInstances(ctx context.Context, poolID string) ([]types.Instance, error) {
	resp, err := a.client.DescribeInstances(ctx, &ec2.DescribeInstancesInput{})
	if err != nil {
		return nil, fmt.Errorf("failed to get instance: %w", err)
	}

	var instances []types.Instance
	for _, reservation := range resp.Reservations {
		instances = append(instances, reservation.Instances...)
	}

	return instances, nil
}

// Create subnet
func (a *AwsCli) CreateSubnet(ctx context.Context, vpcID string, cidr string, region string) (string, error) {
	resp, err := a.client.CreateSubnet(ctx, &ec2.CreateSubnetInput{
		CidrBlock:        aws.String(cidr),
		VpcId:            aws.String(vpcID),
		AvailabilityZone: aws.String(region),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create subnet: %w", err)
	}

	//Tag the subnet with GARM-SUBNET tag
	_, err = a.client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*resp.Subnet.SubnetId},
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("GARM-SUBNET"),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to tag subnet: %w", err)
	}
	return *resp.Subnet.SubnetId, nil
}

func (a *AwsCli) CreateSecurityGroup(ctx context.Context, vpcID string, spec *spec.RunnerSpec) (string, error) {
	resp, err := a.client.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		GroupName:   aws.String("GARM-SG-" + vpcID),
		Description: aws.String("GARM-SG"),
		VpcId:       aws.String(vpcID),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create security group: %w", err)
	}

	//Tag the security group with GARM-SG tag
	_, err = a.client.CreateTags(ctx, &ec2.CreateTagsInput{
		Resources: []string{*resp.GroupId},
		Tags: []types.Tag{
			{
				Key:   aws.String("Name"),
				Value: aws.String("GARM-SG"),
			},
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to tag security group: %w", err)
	}

	rules := spec.SecurityRules()
	_, err = a.client.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:       resp.GroupId,
		IpPermissions: rules,
	})
	if err != nil {
		return "", fmt.Errorf("failed to authorize security group ingress: %w", err)
	}

	return *resp.GroupId, nil
}

func (a *AwsCli) DeleteSecurityGroup(ctx context.Context, vpcID string) error {
	_, err := a.client.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupName: aws.String("GARM-SG-" + vpcID),
	})
	if err != nil {
		return fmt.Errorf("failed to delete security group: %w", err)
	}

	return nil
}

// TODO: Find a better way to implement this
func (a *AwsCli) CreateRunningInstance(ctx context.Context, spec *spec.RunnerSpec, subnetID, groupID string) (string, error) {

	if spec == nil {
		return "", fmt.Errorf("invalid nil runner spec")
	}

	resp, err := a.client.RunInstances(ctx, &ec2.RunInstancesInput{
		ImageId:        aws.String(spec.BootstrapParams.Image),
		InstanceType:   types.InstanceType(spec.BootstrapParams.Flavor),
		MaxCount:       aws.Int32(spec.MaxCount),
		MinCount:       aws.Int32(spec.MinCount),
		SubnetId:       aws.String(subnetID),
		SecurityGroups: []string{groupID},
		UserData:       aws.String(spec.UserData),
	})
	if err != nil {
		return "", fmt.Errorf("failed to create instance: %w", err)
	}

	return *resp.Instances[0].InstanceId, nil
}
