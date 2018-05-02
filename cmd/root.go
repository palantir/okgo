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
	"io/ioutil"

	godelconfig "github.com/palantir/godel/framework/godel/config"
	"github.com/palantir/godel/framework/pluginapi"
	"github.com/palantir/pkg/cobracli"
	"github.com/palantir/pkg/matcher"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
	"gopkg.in/yaml.v2"

	"github.com/palantir/okgo/checker"
	"github.com/palantir/okgo/checker/checkerfactory"
	"github.com/palantir/okgo/okgo"
	"github.com/palantir/okgo/okgo/config"
)

var (
	debugFlagVal           bool
	projectDirFlagVal      string
	okgoConfigFileFlagVal  string
	godelConfigFileFlagVal string
	assetsFlagVal          []string

	cliCheckerFactory okgo.CheckerFactory
)

var rootCmd = &cobra.Command{
	Use: "okgo",
}

func Execute() int {
	return cobracli.ExecuteWithDebugVarAndDefaultParams(rootCmd, &debugFlagVal)
}

func InitAssetCmds(args []string) error {
	if _, _, err := rootCmd.Traverse(args); err != nil && err != pflag.ErrHelp {
		return errors.Wrapf(err, "failed to parse arguments")
	}

	// load checker assets
	checkerCreators, configUpgraders, err := checker.AssetCheckerCreators(assetsFlagVal...)
	if err != nil {
		return err
	}
	cliCheckerFactory, err = checkerfactory.New(checkerCreators, configUpgraders)
	if err != nil {
		return err
	}

	// add run commands based on assets
	addRunSubcommands()

	return nil
}

func init() {
	pluginapi.AddDebugPFlagPtr(rootCmd.PersistentFlags(), &debugFlagVal)
	pluginapi.AddProjectDirPFlagPtr(rootCmd.PersistentFlags(), &projectDirFlagVal)
	pluginapi.AddConfigPFlagPtr(rootCmd.PersistentFlags(), &okgoConfigFileFlagVal)
	pluginapi.AddGodelConfigPFlagPtr(rootCmd.PersistentFlags(), &godelConfigFileFlagVal)
	pluginapi.AddAssetsPFlagPtr(rootCmd.PersistentFlags(), &assetsFlagVal)
}

func okgoProjectParamFromFlags() (okgo.ProjectParam, matcher.Matcher, error) {
	return okgoProjectParamFromVals(okgoConfigFileFlagVal, godelConfigFileFlagVal, cliCheckerFactory)
}

func okgoProjectParamFromVals(okgoConfigFile, godelConfigFile string, factory okgo.CheckerFactory) (okgo.ProjectParam, matcher.Matcher, error) {
	var okgoCfg config.ProjectConfig
	if okgoConfigFile != "" {
		cfg, err := loadConfigFromFile(okgoConfigFile)
		if err != nil {
			return okgo.ProjectParam{}, nil, err
		}
		okgoCfg = cfg
	}
	var godelExcludes matcher.Matcher
	if godelConfigFile != "" {
		cfg, err := godelconfig.ReadGodelConfigFromFile(godelConfigFile)
		if err != nil {
			return okgo.ProjectParam{}, nil, err
		}
		godelExcludes = cfg.Exclude.Matcher()
		okgoCfg.Exclude.Add(cfg.Exclude)
	}
	projectParam, err := okgoCfg.ToParam(factory)
	if err != nil {
		return okgo.ProjectParam{}, nil, err
	}
	if godelExcludes == nil {
		return projectParam, nil, nil
	}
	return projectParam, godelExcludes, nil
}

func loadConfigFromFile(cfgFile string) (config.ProjectConfig, error) {
	cfgBytes, err := ioutil.ReadFile(cfgFile)
	if err != nil {
		return config.ProjectConfig{}, errors.Wrapf(err, "failed to read configuration file")
	}

	upgradedCfg, err := config.UpgradeConfig(cfgBytes, cliCheckerFactory)
	if err != nil {
		return config.ProjectConfig{}, err
	}

	var cfg config.ProjectConfig
	if err := yaml.Unmarshal(upgradedCfg, &cfg); err != nil {
		return config.ProjectConfig{}, errors.Wrapf(err, "failed to unmarshal configuration")
	}
	return cfg, nil
}
