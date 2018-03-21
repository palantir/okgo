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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/palantir/okgo/okgo"
)

var (
	runCheckCmd = &cobra.Command{
		Use:   "run-check",
		Short: "Runs a specific check",
	}
)

func init() {
	RootCmd.AddCommand(runCheckCmd)
}

func addRunSubcommands() {
	for _, checkerType := range cliCheckerFactory.Types() {
		runCheckCmd.AddCommand(createSingleRunCmd(checkerType, cliCheckerFactory))
	}
}

func createSingleRunCmd(checkerType okgo.CheckerType, factory okgo.CheckerFactory) *cobra.Command {
	checker, err := factory.NewChecker(checkerType, nil)
	if err != nil {
		panic(errors.Wrapf(err, "failed to create command for checker type %s", checkerType))
	}
	return &cobra.Command{
		Use: string(checkerType),
		Run: func(cmd *cobra.Command, args []string) {
			checker.RunCheckCmd(args, cmd.OutOrStdout())
		},
	}
}
