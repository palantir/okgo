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
	"encoding/json"
	"fmt"

	"github.com/palantir/godel/framework/pluginapi"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/palantir/okgo/okgo"
)

func AssetRootCmd(creator Creator, upgradeConfigFn pluginapi.UpgradeConfigFn, short string) *cobra.Command {
	checkerType := creator.Type()
	rootCmd := &cobra.Command{
		Use:   string(checkerType),
		Short: short,
	}

	creatorFn := creator.Creator()
	rootCmd.AddCommand(newTypeCmd(checkerType))
	rootCmd.AddCommand(newPriorityCmd(creator.Priority()))
	rootCmd.AddCommand(newVerifyConfigCmd(creatorFn))
	rootCmd.AddCommand(newCheckCmd(creatorFn))
	rootCmd.AddCommand(newRunCheckCmdCmd(creatorFn))
	rootCmd.AddCommand(pluginapi.CobraUpgradeConfigCmd(upgradeConfigFn))

	return rootCmd
}

const typeCmdName = "type"

func newTypeCmd(checkerType okgo.CheckerType) *cobra.Command {
	return &cobra.Command{
		Use:   typeCmdName,
		Short: "Print the type of the checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, err := json.Marshal(checkerType)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal output as JSON")
			}
			cmd.Print(string(outputJSON))
			return nil
		},
	}
}

const priorityCmdName = "priority"

func newPriorityCmd(priority okgo.CheckerPriority) *cobra.Command {
	return &cobra.Command{
		Use:   priorityCmdName,
		Short: "Print the priority of the checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			outputJSON, err := json.Marshal(priority)
			if err != nil {
				return errors.Wrapf(err, "failed to marshal output as JSON")
			}
			cmd.Print(string(outputJSON))
			return nil
		},
	}
}

const commonCmdConfigYMLFlagName = "config-yml"

const (
	verifyConfigCmdName = "verify-config"
)

func newVerifyConfigCmd(creatorFn CreatorFunction) *cobra.Command {
	var configYMLFlagVal string
	verifyConfigCmd := &cobra.Command{
		Use:   verifyConfigCmdName,
		Short: "Verify that the provided input is valid configuration YML for this checker",
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := creatorFn([]byte(configYMLFlagVal))
			return err
		},
	}
	verifyConfigCmd.Flags().StringVar(&configYMLFlagVal, commonCmdConfigYMLFlagName, "", "configuration YML to verify")
	mustMarkFlagsRequired(verifyConfigCmd, commonCmdConfigYMLFlagName)
	return verifyConfigCmd
}

const (
	checkCmdName = "check"
)

func newCheckCmd(creatorFn CreatorFunction) *cobra.Command {
	var (
		configYMLFlagVal  string
		projectDirFlagVal string
	)
	checkCmd := &cobra.Command{
		Use:   fmt.Sprintf("%s [packages]", checkCmdName),
		Short: "Runs the specified check on the provided packages",
		RunE: func(cmd *cobra.Command, args []string) error {
			checker, err := creatorFn([]byte(configYMLFlagVal))
			if err != nil {
				return err
			}
			checker.Check(args, projectDirFlagVal, cmd.OutOrStdout())
			return nil
		},
	}
	checkCmd.Flags().StringVar(&configYMLFlagVal, commonCmdConfigYMLFlagName, "", "YML of Checker configuration")
	checkCmd.Flags().StringVar(&projectDirFlagVal, pluginapi.ProjectDirFlagName, "", "project directory")
	mustMarkFlagsRequired(checkCmd, commonCmdConfigYMLFlagName)
	return checkCmd
}

const (
	runCheckCmdCmdName = "run-check-cmd"
)

func newRunCheckCmdCmd(creatorFn CreatorFunction) *cobra.Command {
	checkCmd := &cobra.Command{
		Use:   runCheckCmdCmdName,
		Short: "Runs the check command using the provided arguments",
		RunE: func(cmd *cobra.Command, args []string) error {
			checker, err := creatorFn(nil)
			if err != nil {
				return err
			}
			checker.RunCheckCmd(args, cmd.OutOrStdout())
			return nil
		},
	}
	return checkCmd
}

func mustMarkFlagsRequired(cmd *cobra.Command, flagNames ...string) {
	for _, currFlagName := range flagNames {
		if err := cmd.MarkFlagRequired(currFlagName); err != nil {
			panic(err)
		}
	}
}
