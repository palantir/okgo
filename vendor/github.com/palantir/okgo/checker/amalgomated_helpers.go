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

package checker

import (
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/kardianos/osext"
	"github.com/palantir/amalgomate/amalgomated"
	"github.com/palantir/godel/framework/pluginapi"
	"github.com/pkg/errors"

	"github.com/palantir/okgo/okgo"
)

type AmalgomatedCheckerParam interface {
	apply(c *amalgomatedChecker)
}

type paramFunc func(*amalgomatedChecker)

func (f paramFunc) apply(c *amalgomatedChecker) {
	f(c)
}

func ParamPriority(priority okgo.CheckerPriority) AmalgomatedCheckerParam {
	return paramFunc(func(c *amalgomatedChecker) {
		c.priority = priority
	})
}

func ParamLineParserWithWd(lineParserWithWd func(line, wd string) okgo.Issue) AmalgomatedCheckerParam {
	return paramFunc(func(c *amalgomatedChecker) {
		c.lineParserWithWd = lineParserWithWd
	})
}

func ParamIncludeProjectDirFlag() AmalgomatedCheckerParam {
	return paramFunc(func(c *amalgomatedChecker) {
		c.includeProjectDirFlag = true
	})
}

func ParamArgs(args ...string) AmalgomatedCheckerParam {
	return paramFunc(func(c *amalgomatedChecker) {
		c.args = args
	})
}

func NewAmalgomatedChecker(typeName okgo.CheckerType, params ...AmalgomatedCheckerParam) okgo.Checker {
	checker := &amalgomatedChecker{
		typeName:         typeName,
		lineParserWithWd: okgo.NewIssueFromLine,
	}
	for _, p := range params {
		if p == nil {
			continue
		}
		p.apply(checker)
	}
	return checker
}

type amalgomatedChecker struct {
	typeName              okgo.CheckerType
	priority              okgo.CheckerPriority
	lineParserWithWd      func(line, wd string) okgo.Issue
	includeProjectDirFlag bool
	args                  []string
}

func (c *amalgomatedChecker) Type() (okgo.CheckerType, error) {
	return c.typeName, nil
}

func (c *amalgomatedChecker) Priority() (okgo.CheckerPriority, error) {
	return c.priority, nil
}

func (c *amalgomatedChecker) Check(pkgPaths []string, projectDir string, stdout io.Writer) {
	var args []string
	if c.includeProjectDirFlag {
		args = append(args, "--"+pluginapi.ProjectDirFlagName, projectDir)
	}
	args = append(args, c.args...)
	args = append(args, pkgPaths...)

	cmd, wd := AmalgomatedCheckCmd(string(c.typeName), args, stdout)
	if cmd == nil {
		return
	}
	RunCommandAndStreamOutput(cmd, func(line string) okgo.Issue {
		return c.lineParserWithWd(line, wd)
	}, stdout)
}

func (c *amalgomatedChecker) RunCheckCmd(args []string, stdout io.Writer) {
	AmalgomatedRunRawCheck(string(c.typeName), args, stdout)
}

func AmalgomatedCheckCmd(amalgomatedCmdName string, args []string, stdout io.Writer) (*exec.Cmd, string) {
	pathToSelf, err := osext.Executable()
	if err != nil {
		okgo.WriteErrorAsIssue(errors.Wrapf(err, "failed to determine path to executable"), stdout)
		return nil, ""
	}

	wd, err := os.Getwd()
	if err != nil {
		okgo.WriteErrorAsIssue(errors.Wrapf(err, "failed to determine working directory"), stdout)
		return nil, ""
	}

	return exec.Command(pathToSelf, append([]string{amalgomated.ProxyCmdPrefix + amalgomatedCmdName}, args...)...), wd
}

func AmalgomatedRunRawCheck(amalgomatedCmdName string, args []string, stdout io.Writer) {
	cmd, _ := AmalgomatedCheckCmd(amalgomatedCmdName, args, stdout)
	if cmd == nil {
		return
	}
	cmd.Stdout = stdout
	cmd.Stderr = stdout
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			fmt.Fprintf(stdout, "command %v failed with error %v\n", cmd.Args, err)
		}
	}
}
