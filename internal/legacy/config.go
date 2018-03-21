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

	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/okgo/okgo"
)

type legacyConfigStruct struct {
	Legacy bool `yaml:"legacy-config"`

	ReleaseTag string                         `yaml:"release-tag"`
	Checks     map[string]legacyCheckerStruct `yaml:"checks"`
	Exclude    matcher.NamesPathsCfg          `yaml:"exclude"`
}

type legacyCheckerStruct struct {
	Skip    bool                 `yaml:"skip"`
	Args    []string             `yaml:"args"`
	Filters []legacyFilterStruct `yaml:"filters"`
}

type legacyFilterStruct struct {
	Type  string `yaml:"type"`
	Value string `yaml:"value"`
}

type legacyAssetConfigStruct struct {
	Legacy bool     `yaml:"legacy-config"`
	Args   []string `yaml:"args"`
}

func IsLegacyConfig(cfgBytes []byte) bool {
	var cfg legacyConfigStruct
	if err := yaml.Unmarshal(cfgBytes, &cfg); err != nil {
		return false
	}
	return cfg.Legacy
}

func UpgradeLegacyConfig(cfgBytes []byte, factory okgo.CheckerFactory) ([]byte, error) {
	var legacyCfg legacyConfigStruct
	if err := yaml.UnmarshalStrict(cfgBytes, &legacyCfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal legacy configuration")
	}

	upgradedCfg, err := upgradeLegacyConfig(legacyCfg, factory)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to upgrade legacy configuration")
	}

	// indicates that this is the default config
	if upgradedCfg == nil {
		return nil, nil
	}

	outputBytes, err := yaml.Marshal(*upgradedCfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal configuration as YAML")
	}
	return outputBytes, nil
}

func upgradeLegacyConfig(legacyCfg legacyConfigStruct, factory okgo.CheckerFactory) (*okgo.ProjectConfig, error) {
	// upgrade top-level configuration
	upgradedCfg := okgo.ProjectConfig{
		ReleaseTag: legacyCfg.ReleaseTag,
		Exclude:    legacyCfg.Exclude,
	}

	// delegate to asset upgraders
	var sortedKeys []string
	for k := range legacyCfg.Checks {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Strings(sortedKeys)

	if len(sortedKeys) > 0 {
		upgradedCfg.Checks = make(map[okgo.CheckerType]okgo.CheckerConfig)
	}

	for _, k := range sortedKeys {
		upgrader, err := factory.ConfigUpgrader(okgo.CheckerType(k))
		if err != nil {
			return nil, err
		}

		assetCfgBytes, err := yaml.Marshal(legacyAssetConfigStruct{
			Legacy: true,
			Args:   legacyCfg.Checks[k].Args,
		})
		if err != nil {
			return nil, errors.Wrapf(err, "failed to marshal legacy asset configuration for check %q as YAML", k)
		}

		upgradedBytes, err := upgrader.UpgradeConfig(assetCfgBytes)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to upgrade configuration for check %q", k)
		}

		var yamlRep yaml.MapSlice
		if err := yaml.Unmarshal(upgradedBytes, &yamlRep); err != nil {
			return nil, errors.Wrapf(err, "failed to unmarshal YAML of upgraded configuration for check %q", k)
		}

		var filters []okgo.FilterConfig
		var excludeConfig matcher.NamesPathsCfg
		for _, legacyFilter := range legacyCfg.Checks[k].Filters {
			switch legacyFilter.Type {
			case "", "message":
				filters = append(filters, okgo.FilterConfig{
					Type:  okgo.FilterType(legacyFilter.Type),
					Value: legacyFilter.Value,
				})
			case "name":
				excludeConfig.Names = append(excludeConfig.Names, legacyFilter.Value)
			case "path":
				excludeConfig.Paths = append(excludeConfig.Paths, legacyFilter.Value)
			}
		}

		upgradedCfg.Checks[okgo.CheckerType(k)] = okgo.CheckerConfig{
			Skip:    legacyCfg.Checks[k].Skip,
			Config:  yamlRep,
			Filters: filters,
			Exclude: excludeConfig,
		}
	}
	return &upgradedCfg, nil
}
