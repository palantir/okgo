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
	"encoding/json"
	"fmt"
	"io"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/pkg/errors"
)

type CheckerPriority int

type Checker interface {
	// Type returns the type of this Checker.
	Type() (CheckerType, error)

	// Priority returns the priority of the check. A lower number indicates a higher priority (will be run earlier).
	Priority() (CheckerPriority, error)

	// Check runs the check on the specified packages and writes the output to the provided io.Writer. All output
	// written to the writer must be the JSON-serialized form of Issue, where there is one issue per line. Note that
	// this function does not return an error -- if an error is encountered during execution, it should be written to
	// the provided io.Writer as an Issue (issues that aren't issues with underlying code should only populate the
	// "Content" field). If any output is written to the writer, the check is considered to have failed.
	Check(pkgPaths []string, projectDir string, stdout io.Writer)

	// RunCheckCmd runs the check command directly using the provided arguments and writes the unaltered output to the
	// provided writer.
	RunCheckCmd(args []string, stdout io.Writer)
}

type CheckerFactory interface {
	AllCheckers() []CheckerType
	NewChecker(checkerType CheckerType, cfgYMLBytes []byte) (Checker, error)
}

// NewIssueFromJSON creates an Issue from the provided input. If the provided input is the JSON representation of an
// Issue, it is decoded and returned. Otherwise, a new Issue is created with the content set to the provided string and
// all other fields zero'd out.
func NewIssueFromJSON(in string) Issue {
	var issue Issue
	if err := json.Unmarshal([]byte(in), &issue); err != nil {
		return Issue{
			Content: in,
		}
	}
	return issue
}

var lineRegexp = regexp.MustCompile(`(.+):(\d+):(\d+): (.+)`)

// NewIssueFromLine creates a new Issue from the provided input. If the provided input matches the regular expression
// `(.+):(\d+):(\d+): (.+)`, then a new issue is created from the corresponding components of the regular expression.
// Otherwise, a new issue is created where the Content of the issue is the entire input. If the Path component of the
// input line is an absolute path, it is converted to a relative path with the provided wd used as the base directory.
func NewIssueFromLine(in, wd string) Issue {
	issue := Issue{
		Content: in,
	}

	matches := lineRegexp.FindStringSubmatch(in)
	if len(matches) == 0 {
		return issue
	}

	issuePath := matches[1]
	if filepath.IsAbs(issuePath) {
		relPath, err := filepath.Rel(wd, matches[1])
		if err != nil {
			return issue
		}
		issuePath = relPath
	}
	lineNum, err := strconv.Atoi(matches[2])
	if err != nil {
		return issue
	}
	colNum, err := strconv.Atoi(matches[3])
	if err != nil {
		return issue
	}

	issue.Path = issuePath
	issue.Line = lineNum
	issue.Col = colNum
	issue.Content = matches[4]
	return issue
}

func WriteErrorAsIssue(err error, stdout io.Writer) {
	issue := Issue{
		Content: err.Error(),
	}
	bytes, err := json.Marshal(issue)
	if err != nil {
		// Issue must support JSON-serialization
		panic(errors.Wrapf(err, "failed to JSON-serialize issue %+v", issue))
	}
	fmt.Fprintln(stdout, string(bytes))
}
