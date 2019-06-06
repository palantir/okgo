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
	"github.com/palantir/pkg/matcher"
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

type CheckerParam struct {
	Skip     bool
	Priority *CheckerPriority
	Checker  Checker
	Filters  []Filter
	Exclude  matcher.Matcher
}

type Filter interface {
	Filter(issue Issue) bool
}
