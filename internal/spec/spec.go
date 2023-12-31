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

package spec

import (
	"encoding/base64"
	"encoding/json"
	"fmt"

	"github.com/cloudbase/garm-provider-aws/config"
	"github.com/cloudbase/garm-provider-common/cloudconfig"
	"github.com/cloudbase/garm-provider-common/params"
	"github.com/cloudbase/garm-provider-common/util"
)

func newExtraSpecsFromBootstrapData(data params.BootstrapInstance) (*extraSpecs, error) {
	spec := &extraSpecs{}

	if len(data.ExtraSpecs) > 0 {
		if err := json.Unmarshal(data.ExtraSpecs, spec); err != nil {
			return nil, fmt.Errorf("failed to unmarshal extra specs: %w", err)
		}
	}
	spec.ensureValidExtraSpec()

	return spec, nil
}

type extraSpecs struct {
	MinCount int32
	MaxCount int32
}

func (e *extraSpecs) ensureValidExtraSpec() {

}

func GetRunnerSpecFromBootstrapParams(cfg config.Config, data params.BootstrapInstance, controllerID string) (*RunnerSpec, error) {
	tools, err := util.GetTools(data.OSType, data.OSArch, data.Tools)
	if err != nil {
		return nil, fmt.Errorf("failed to get tools: %s", err)
	}

	extraSpecs, err := newExtraSpecsFromBootstrapData(data)
	if err != nil {
		return nil, fmt.Errorf("error loading extra specs: %w", err)
	}

	spec := &RunnerSpec{
		Region:          cfg.Region,
		Tools:           tools,
		BootstrapParams: data,
		MinCount:        1,
		MaxCount:        1,
	}

	spec.MergeExtraSpecs(extraSpecs)
	spec.SetUserData()

	return spec, nil
}

type RunnerSpec struct {
	Region          string
	Tools           params.RunnerApplicationDownload
	BootstrapParams params.BootstrapInstance
	UserData        string
	MinCount        int32
	MaxCount        int32
}

func (r *RunnerSpec) Validate() error {
	if r.Region == "" {
		return fmt.Errorf("missing region")
	}
	if r.BootstrapParams.Name == "" {
		return fmt.Errorf("missing bootstrap params")
	}
	return nil
}

func (r *RunnerSpec) MergeExtraSpecs(extraSpecs *extraSpecs) {
	if extraSpecs.MinCount > 1 {
		r.MinCount = extraSpecs.MinCount
	}
	if extraSpecs.MaxCount > 1 {
		r.MaxCount = extraSpecs.MaxCount
	}
}

func (r *RunnerSpec) SetUserData() error {
	customData, err := r.ComposeUserData()
	if err != nil {
		return fmt.Errorf("failed to compose userdata: %w", err)
	}

	if len(customData) == 0 {
		return fmt.Errorf("failed to generate custom data")
	}

	asBase64 := base64.StdEncoding.EncodeToString(customData)
	r.UserData = asBase64
	return nil
}

func (r *RunnerSpec) ComposeUserData() ([]byte, error) {
	switch r.BootstrapParams.OSType {
	case params.Linux, params.Windows:
		udata, err := cloudconfig.GetCloudConfig(r.BootstrapParams, r.Tools, r.BootstrapParams.Name)
		if err != nil {
			return nil, fmt.Errorf("failed to generate userdata: %w", err)
		}
		return []byte(udata), nil
	}
	return nil, fmt.Errorf("unsupported OS type for cloud config: %s", r.BootstrapParams.OSType)
}
