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
	"regexp"

	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v2"
)

type ProjectParam struct {
	ReleaseTag string
	Checks     map[CheckerType]CheckerParam
}

type CheckerType string

type ByCheckerType []CheckerType

func (a ByCheckerType) Len() int           { return len(a) }
func (a ByCheckerType) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }
func (a ByCheckerType) Less(i, j int) bool { return a[i] < a[j] }

type ProjectConfig struct {
	// ReleaseTag specifies the newest Go release build tag supported by the codebase being checked. If this value
	// is not specified, it defaults to the Go release that was used to build the check tool. If the code being
	// checked is known to use a version of Go that is earlier than the version of Go used to build the check tool
	// and the codebase being checked contains build tags for the newer Go version, this value should be explicitly
	// set. For example, if the check tool was compiled using Go 1.8 but the codebase being checked uses Go 1.7 and
	// contains files that use the "// +build go1.8" build tag, then this should be set to "go1.7".
	ReleaseTag string `yaml:"release-tag"`

	// Checks specifies the configuration used by the checks. The key is the name of the check and the value is the
	// custom configuration for that check.
	Checks map[CheckerType]CheckerConfig `yaml:"checks"`

	// Exclude specifies the paths that should be excluded from all checks.
	Exclude matcher.NamesPathsCfg `yaml:"exclude"`
}

func (c *ProjectConfig) ToParam(factory CheckerFactory) (ProjectParam, error) {
	var checks map[CheckerType]CheckerParam
	if len(c.Checks) > 0 {
		checks = make(map[CheckerType]CheckerParam)
		for k, v := range c.Checks {
			currParam, err := v.ToParam(k, factory, c.Exclude)
			if err != nil {
				return ProjectParam{}, err
			}
			checks[k] = currParam
		}
	}
	return ProjectParam{
		ReleaseTag: c.ReleaseTag,
		Checks:     checks,
	}, nil
}

type CheckerParam struct {
	Skip    bool
	Checker Checker
	Filters []Filter
	Exclude matcher.Matcher
}

type CheckerConfig struct {
	// Skip indicates whether or not the check should be skipped entirely.
	Skip bool `yaml:"skip"`

	// Config is the YAML configuration content for the Checker.
	Config yaml.MapSlice `yaml:"config"`

	// Filters specifies the filter definitions. Raw output lines that match the filter are excluded from
	// processing.
	Filters []FilterConfig `yaml:"filters"`

	// Exclude specifies the paths that should be excluded from this check.
	Exclude matcher.NamesPathsCfg `yaml:"exclude"`
}

func (c *CheckerConfig) ToParam(checkerType CheckerType, factory CheckerFactory, globalExclude matcher.NamesPathsCfg) (CheckerParam, error) {
	checker, err := newChecker(checkerType, c.Config, factory)
	if err != nil {
		return CheckerParam{}, err
	}
	var filters []Filter
	for _, filterCfg := range c.Filters {
		currFilter, err := filterCfg.ToFilter()
		if err != nil {
			return CheckerParam{}, err
		}
		filters = append(filters, currFilter)
	}
	return CheckerParam{
		Skip:    c.Skip,
		Checker: checker,
		Filters: filters,
		Exclude: matcher.Any(c.Exclude.Matcher(), globalExclude.Matcher()),
	}, nil
}

func newChecker(checkerType CheckerType, cfgYML yaml.MapSlice, factory CheckerFactory) (Checker, error) {
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

type Filter interface {
	Filter(issue Issue) bool
}

type FilterConfig struct {
	// Type specifies the type of the filter.
	Type FilterType `yaml:"type"`

	// Value is the value of the filter.
	Value string `yaml:"value"`
}

func (f *FilterConfig) ToFilter() (Filter, error) {
	switch f.Type {
	case "", MessageFilterType:
		return newMessageFilter(f.Value)
	}
	return nil, errors.Errorf("unrecognized filter type %s", f.Type)
}

type FilterType string

const (
	MessageFilterType FilterType = "message"
)

func newMessageFilter(input string) (Filter, error) {
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

func (f *messageFilterImpl) Filter(issue Issue) bool {
	return f.msgRegexp.MatchString(issue.Content)
}
