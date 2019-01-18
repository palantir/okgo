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
	"fmt"
)

type Issue struct {
	Path    string `json:"path"`
	Line    int    `json:"line"`
	Col     int    `json:"col"`
	Content string `json:"content"`
}

func (issue *Issue) String() string {
	var output string
	if issue.Path != "" {
		output += fmt.Sprintf("%s:", issue.Path)
	}
	if issue.Line != 0 {
		output += fmt.Sprintf("%d:", issue.Line)
	}
	if issue.Col != 0 {
		output += fmt.Sprintf("%d:", issue.Col)
	}
	if issue.Content != "" {
		if output != "" {
			output += " "
		}
		output += issue.Content
	}
	return output
}
