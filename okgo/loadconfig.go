// Copyright 2016 Palantir Technologies, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package okgo

import (
	"io/ioutil"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

func LoadConfigFromFile(cfgFile string) (ProjectConfig, error) {
	cfgBytes, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return ProjectConfig{}, errors.Wrapf(err, "failed to read configuration file")
	}
	return LoadConfig(cfgBytes)
}

func LoadConfig(cfgBytes []byte) (ProjectConfig, error) {
	var cfg ProjectConfig
	if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
		return ProjectConfig{}, errors.Wrapf(err, "failed to unmarshal configuration")
	}
	return cfg, nil
}
