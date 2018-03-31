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

package v0

import (
	"bytes"
	"sort"

	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/okgo/okgo"
)

type ProjectConfig struct {
	// ReleaseTag specifies the newest Go release build tag supported by the codebase being checked. If this value
	// is not specified, it defaults to the Go release that was used to build the check tool. If the code being
	// checked is known to use a version of Go that is earlier than the version of Go used to build the check tool
	// and the codebase being checked contains build tags for the newer Go version, this value should be explicitly
	// set. For example, if the check tool was compiled using Go 1.8 but the codebase being checked uses Go 1.7 and
	// contains files that use the "// +build go1.8" build tag, then this should be set to "go1.7".
	ReleaseTag string `yaml:"release-tag,omitempty"`

	// Checks specifies the configuration used by the checks. The key is the name of the check and the value is the
	// custom configuration for that check.
	Checks map[okgo.CheckerType]CheckerConfig `yaml:"checks,omitempty"`

	// Exclude specifies the paths that should be excluded from all checks.
	Exclude matcher.NamesPathsCfg `yaml:"exclude,omitempty"`
}

type CheckerConfig struct {
	// Skip indicates whether or not the check should be skipped entirely.
	Skip bool `yaml:"skip,omitempty"`

	// Priority is the priority for this check. If the value is non-nil, this value is used instead of the priority
	// provided by the checker.
	Priority *int `yaml:"priority,omitempty"`

	// Config is the YAML configuration content for the Checker.
	Config yaml.MapSlice `yaml:"config,omitempty"`

	// Filters specifies the filter definitions. Raw output lines that match the filter are excluded from
	// processing.
	Filters []FilterConfig `yaml:"filters,omitempty"`

	// Exclude specifies the paths that should be excluded from this check.
	Exclude matcher.NamesPathsCfg `yaml:"exclude,omitempty"`
}

type FilterConfig struct {
	// Type specifies the type of the filter.
	Type FilterType `yaml:"type,omitempty"`

	// Value is the value of the filter.
	Value string `yaml:"value,omitempty"`
}

type FilterType string

const (
	MessageFilterType FilterType = "message"
)

func UpgradeConfig(cfgBytes []byte, factory okgo.CheckerFactory) ([]byte, error) {
	var cfg ProjectConfig
	if err := yaml.UnmarshalStrict(cfgBytes, &cfg); err != nil {
		return nil, errors.Wrapf(err, "failed to unmarshal check-plugin v0 configuration")
	}
	changed, err := upgradeAssets(&cfg, factory)
	if err != nil {
		return nil, err
	}
	if !changed {
		return cfgBytes, nil
	}
	upgradedBytes, err := yaml.Marshal(cfg)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal check-plugin v0 configuration")
	}
	return upgradedBytes, nil
}

// upgradeAssets upgrades the assets for the provided configuration. Returns true if any upgrade operations were
// performed. If any upgrade operations were performed, the provided configuration is modified directly.
func upgradeAssets(cfg *ProjectConfig, factory okgo.CheckerFactory) (changed bool, rErr error) {
	var sortedKeys []okgo.CheckerType
	for k := range cfg.Checks {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(okgo.ByCheckerType(sortedKeys))

	for _, k := range sortedKeys {
		upgrader, err := factory.ConfigUpgrader(k)
		if err != nil {
			return false, err
		}

		assetCfgBytes, err := yaml.Marshal(cfg.Checks[k].Config)
		if err != nil {
			return false, errors.Wrapf(err, "failed to marshal check %q configuration", k)
		}

		upgradedBytes, err := upgrader.UpgradeConfig(assetCfgBytes)
		if err != nil {
			return false, errors.Wrapf(err, "failed to upgrade check %q configuration", k)
		}

		if bytes.Equal(assetCfgBytes, upgradedBytes) {
			// upgrade was a no-op: do not modify configuration and continue
			continue
		}
		changed = true

		var yamlRep yaml.MapSlice
		if err := yaml.Unmarshal(upgradedBytes, &yamlRep); err != nil {
			return false, errors.Wrapf(err, "failed to unmarshal check %q configuration as yaml.MapSlice", k)
		}

		// update configuration for asset in original configuration
		assetCheckCfg := cfg.Checks[k]
		assetCheckCfg.Config = yamlRep
		cfg.Checks[k] = assetCheckCfg
	}
	return changed, nil
}
