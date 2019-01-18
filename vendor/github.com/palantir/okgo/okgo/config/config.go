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

package config

import (
	"regexp"

	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"

	"github.com/palantir/okgo/okgo"
	"github.com/palantir/okgo/okgo/config/internal/v0"
)

type ProjectConfig v0.ProjectConfig

func (c *ProjectConfig) ToParam(factory okgo.CheckerFactory) (okgo.ProjectParam, error) {
	var checks map[okgo.CheckerType]okgo.CheckerParam

	allCheckerConfigs := make(map[okgo.CheckerType]CheckerConfig)
	// populate default configuration for all checks (contains only excludes)
	for _, checkerType := range factory.Types() {
		allCheckerConfigs[checkerType] = CheckerConfig{
			Exclude: c.Exclude,
		}
	}
	// populate provided configurations
	for k, v := range c.Checks {
		allCheckerConfigs[k] = CheckerConfig(v)
	}

	// create parameters
	if len(allCheckerConfigs) > 0 {
		checks = make(map[okgo.CheckerType]okgo.CheckerParam)
		for k, v := range allCheckerConfigs {
			currParam, err := v.ToParam(k, factory, c.Exclude)
			if err != nil {
				return okgo.ProjectParam{}, err
			}
			checks[k] = currParam
		}
	}

	return okgo.ProjectParam{
		Checks: checks,
	}, nil
}

type CheckerConfig v0.CheckerConfig

func (c *CheckerConfig) ToParam(checkerType okgo.CheckerType, factory okgo.CheckerFactory, globalExclude matcher.NamesPathsCfg) (okgo.CheckerParam, error) {
	checker, err := newChecker(checkerType, c.Config, factory)
	if err != nil {
		return okgo.CheckerParam{}, err
	}
	var filters []okgo.Filter
	for _, filterCfg := range c.Filters {
		currFilter, err := (*FilterConfig)(&filterCfg).ToFilter()
		if err != nil {
			return okgo.CheckerParam{}, err
		}
		filters = append(filters, currFilter)
	}
	combinedExcludeConfig := c.Exclude
	combinedExcludeConfig.Add(globalExclude)
	return okgo.CheckerParam{
		Skip:     c.Skip,
		Priority: (*okgo.CheckerPriority)(c.Priority),
		Checker:  checker,
		Filters:  filters,
		Exclude:  combinedExcludeConfig.Matcher(),
	}, nil
}

func newChecker(checkerType okgo.CheckerType, cfgYML yaml.MapSlice, factory okgo.CheckerFactory) (okgo.Checker, error) {
	if checkerType == "" {
		return nil, errors.Errorf("checkerType must be non-empty")
	}
	if factory == nil {
		return nil, errors.Errorf("factory must be provided")
	}
	cfgYMLBytes, err := yaml.Marshal(cfgYML)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to marshal configuration")
	}
	return factory.NewChecker(checkerType, cfgYMLBytes)
}

type FilterConfig v0.FilterConfig

func (f *FilterConfig) ToFilter() (okgo.Filter, error) {
	var filterCreator func(string) (okgo.Filter, error)
	switch f.Type {
	case "", v0.MessageFilterType:
		filterCreator = newMessageFilter
	default:
		return nil, errors.Errorf("unrecognized filter type %s", f.Type)
	}
	return filterCreator(f.Value)
}

func newMessageFilter(input string) (okgo.Filter, error) {
	msgRegexp, err := regexp.Compile(input)
	if err != nil {
		return nil, errors.WithStack(err)
	}
	return &messageFilterImpl{
		msgRegexp: msgRegexp,
	}, nil
}

type messageFilterImpl struct {
	msgRegexp *regexp.Regexp
}

func (f *messageFilterImpl) Filter(issue okgo.Issue) bool {
	return f.msgRegexp.MatchString(issue.Content)
}
