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

package legacy

import (
	"sort"

	"github.com/palantir/godel/pkg/versionedconfig"
	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/okgo/okgo"
	"github.com/palantir/okgo/okgo/config/internal/v0"
)

type ProjectConfig struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline"`

	ReleaseTag string                             `yaml:"release-tag"`
	Checks     map[okgo.CheckerType]CheckerConfig `yaml:"checks"`
	Exclude    matcher.NamesPathsCfg              `yaml:"exclude"`
}

type CheckerConfig struct {
	Skip    bool           `yaml:"skip"`
	Args    []string       `yaml:"args"`
	Filters []FilterConfig `yaml:"filters"`
}

type FilterConfig struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

type AssetConfig struct {
	versionedconfig.ConfigWithLegacy `yaml:",inline"`

	Args []string `yaml:"args"`
}

func UpgradeConfig(cfgBytes []byte, factory okgo.CheckerFactory) ([]byte, error) {
	var legacyCfg ProjectConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &legacyCfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal check-plugin legacy configuration")
	}

	v0Cfg, err := upgradeLegacyConfig(legacyCfg, factory)
	if err != nil {
		return nil, errors.WithStack(err)
	}

	// indicates that this is the default config
	if v0Cfg == nil {
		return nil, nil
	}

	outputBytes, err := yaml.Marshal(*v0Cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal check-plugin v0 configuration")
	}
	return outputBytes, nil
}

func upgradeLegacyConfig(legacyCfg ProjectConfig, factory okgo.CheckerFactory) (*v0.ProjectConfig, error) {
	// upgrade top-level configuration
	upgradedCfg := v0.ProjectConfig{
		ReleaseTag: legacyCfg.ReleaseTag,
		Exclude:    legacyCfg.Exclude,
	}

	// delegate to asset upgraders
	var sortedKeys []okgo.CheckerType
	for k := range legacyCfg.Checks {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(okgo.ByCheckerType(sortedKeys))

	if len(sortedKeys) > 0 {
		upgradedCfg.Checks = make(map[okgo.CheckerType]v0.CheckerConfig)
	}

	for _, k := range sortedKeys {
		upgrader, err := factory.ConfigUpgrader(k)
		if err != nil {
			return nil, err
		}

		assetCfgBytes, err := yaml.Marshal(AssetConfig{
			ConfigWithLegacy: versionedconfig.ConfigWithLegacy{
				Legacy: true,
			},
			Args: legacyCfg.Checks[k].Args,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal check %q legacy configuration", k)
		}

		upgradedBytes, err := upgrader.UpgradeConfig(assetCfgBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to upgrade check %q legacy configuration", k)
		}

		var yamlRep yaml.MapSlice
		if err := yaml.Unmarshal(upgradedBytes, &yamlRep); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal check %q configuration as yaml.MapSlice", k)
		}

		var filters []v0.FilterConfig
		var excludeConfig matcher.NamesPathsCfg
		for _, legacyFilter := range legacyCfg.Checks[k].Filters {
			switch legacyFilter.Type {
			case "", "message":
				filters = append(filters, v0.FilterConfig{
					Type:  v0.FilterType(legacyFilter.Type),
					Value: legacyFilter.Value,
				})
			case "name":
				excludeConfig.Names = append(excludeConfig.Names, legacyFilter.Value)
			case "path":
				excludeConfig.Paths = append(excludeConfig.Paths, legacyFilter.Value)
			}
		}

		upgradedCfg.Checks[k] = v0.CheckerConfig{
			Skip:    legacyCfg.Checks[k].Skip,
			Config:  yamlRep,
			Filters: filters,
			Exclude: excludeConfig,
		}
	}
	return &upgradedCfg, nil
}
