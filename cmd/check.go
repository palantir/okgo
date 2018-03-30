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

package cmd

import (
	"os"
	"path"
	"path/filepath"
	"runtime"

	"github.com/palantir/pkg/matcher"
	"github.com/palantir/pkg/pkgpath"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/palantir/okgo/okgo"
	"github.com/palantir/okgo/okgo/check"
)

var (
	checkCmd = &cobra.Command{
		Use:   "check [flags] [checks]",
		Short: "Run checks (runs all checks if none are specified)",
		RunE: func(cmd *cobra.Command, args []string) error {
			projectParam, godelExcludeMatcher, err := okgoProjectParamFromFlags()
			if err != nil {
				return err
			}
			parallelism := 1
			if parallelFlagVal {
				parallelism = runtime.NumCPU()
			}
			pkgs, err := pkgsInProject(projectDirFlagVal, godelExcludeMatcher)
			if err != nil {
				return err
			}
			checkerTypes, err := toCheckerTypes(args, cliCheckerFactory)
			if err != nil {
				return err
			}
			return check.Run(projectParam, checkerTypes, pkgs, projectDirFlagVal, cliCheckerFactory, parallelism, cmd.OutOrStdout())
		},
	}

	parallelFlagVal bool
)

func pkgsInProject(projectDir string, exclude matcher.Matcher) ([]string, error) {
	wd, err := os.Getwd()
	if err != nil {
		return nil, errors.Wrapf(err, "failed to determine working directory")
	}
	if !filepath.IsAbs(projectDir) {
		projectDir = path.Join(wd, projectDir)
	}
	var relPathPrefix string
	if wd != projectDir {
		relPathPrefixVal, err := filepath.Rel(wd, projectDir)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to determine relative path")
		}
		relPathPrefix = relPathPrefixVal
	}
	pkgs, err := pkgpath.PackagesInDir(projectDir, exclude)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to list packages")
	}
	pkgPaths, err := pkgs.Paths(pkgpath.Relative)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get package paths")
	}
	if relPathPrefix != "" {
		for i, pkgPath := range pkgPaths {
			pkgPaths[i] = "./" + path.Join(relPathPrefix, pkgPath)
		}
	}
	return pkgPaths, nil
}

func toCheckerTypes(in []string, factory okgo.CheckerFactory) ([]okgo.CheckerType, error) {
	allCheckers := factory.Types()
	if len(in) == 0 {
		return allCheckers, nil
	}

	checkerMap := make(map[string]okgo.CheckerType)
	for _, k := range allCheckers {
		checkerMap[string(k)] = k
	}
	var out []okgo.CheckerType
	var unknown []string
	for _, currIn := range in {
		checker, ok := checkerMap[currIn]
		if !ok {
			unknown = append(unknown, currIn)
			continue
		}
		out = append(out, checker)
	}
	if len(unknown) > 0 {
		return nil, errors.Errorf("provided checker type(s) %v not valid: valid values are %v", unknown, allCheckers)
	}
	return out, nil
}

func init() {
	checkCmd.Flags().BoolVar(&parallelFlagVal, "parallel", true, "run checks in parallel")

	RootCmd.AddCommand(checkCmd)
}
