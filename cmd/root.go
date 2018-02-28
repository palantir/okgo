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
	"path"

	"github.com/palantir/godel/framework/godellauncher"
	"github.com/palantir/godel/framework/pluginapi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"

	"github.com/palantir/okgo/checker"
	"github.com/palantir/okgo/okgo"
)

var (
	projectDirFlagVal      string
	okgoConfigFileFlagVal  string
	godelConfigFileFlagVal string
	assetsFlagVal          []string

	cliCheckerFactory okgo.CheckerFactory
)

var RootCmd = &cobra.Command{
	Use: "okgo",
}

func InitAssetCmds(args []string) error {
	if _, _, err := RootCmd.Traverse(args); err != nil && err != pflag.ErrHelp {
		return errors.Wrapf(err, "failed to parse arguments 2")
	}

	// load checker assets
	checkerCreators, err := checker.AssetCheckerCreators(assetsFlagVal...)
	if err != nil {
		return err
	}
	cliCheckerFactory, err = checker.NewCheckerFactory(checkerCreators...)
	if err != nil {
		return err
	}

	// add run commands based on assets
	addRunSubcommands()

	return nil
}

func init() {
	pluginapi.AddProjectDirPFlagPtr(RootCmd.PersistentFlags(), &projectDirFlagVal)
	pluginapi.AddConfigPFlagPtr(RootCmd.PersistentFlags(), &okgoConfigFileFlagVal)
	pluginapi.AddGodelConfigPFlagPtr(RootCmd.PersistentFlags(), &godelConfigFileFlagVal)
	pluginapi.AddAssetsPFlagPtr(RootCmd.PersistentFlags(), &assetsFlagVal)
}

func okgoProjectParamFromFlags() (okgo.ProjectParam, error) {
	return okgoProjectParamFromVals(okgoConfigFileFlagVal, godelConfigFileFlagVal, cliCheckerFactory)
}

func okgoProjectParamFromVals(okgoConfigFile, godelConfigFile string, factory okgo.CheckerFactory) (okgo.ProjectParam, error) {
	var okgoCfg okgo.ProjectConfig
	if okgoConfigFile != "" {
		cfg, err := okgo.LoadConfigFromFile(okgoConfigFile)
		if err != nil {
			return okgo.ProjectParam{}, err
		}
		okgoCfg = cfg
	}
	if godelConfigFile != "" {
		cfg, err := godellauncher.ReadGodelConfig(path.Dir(godelConfigFile))
		if err != nil {
			return okgo.ProjectParam{}, err
		}
		okgoCfg.Exclude.Add(cfg.Exclude)
	}
	projectParam, err := okgoCfg.ToParam(factory)
	if err != nil {
		return okgo.ProjectParam{}, err
	}
	return projectParam, nil
}
