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
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sort"

	"github.com/pkg/errors"

	"github.com/palantir/okgo/okgo"
)

type CreatorFunction func(cfgYML []byte) (okgo.Checker, error)

type Creator interface {
	Type() okgo.CheckerType
	Priority() okgo.CheckerPriority
	Creator() CreatorFunction
}

type creatorStruct struct {
	checkerType okgo.CheckerType
	priority    okgo.CheckerPriority
	creator     CreatorFunction
}

func (c *creatorStruct) Type() okgo.CheckerType {
	return c.checkerType
}

func (c *creatorStruct) Priority() okgo.CheckerPriority {
	return c.priority
}

func (c *creatorStruct) Creator() CreatorFunction {
	return c.creator
}

func NewCreator(checkerType okgo.CheckerType, priority okgo.CheckerPriority, creatorFn CreatorFunction) Creator {
	return &creatorStruct{
		checkerType: checkerType,
		priority:    priority,
		creator:     creatorFn,
	}
}

func AssetCheckerCreators(assetPaths ...string) ([]Creator, []okgo.ConfigUpgrader, error) {
	var checkerCreators []Creator
	var configUpgraders []okgo.ConfigUpgrader
	checkerTypeToAssets := make(map[okgo.CheckerType][]string)
	for _, currAssetPath := range assetPaths {
		currAssetPath := currAssetPath
		currChecker := assetChecker{
			assetPath: currAssetPath,
		}
		checkerType, err := currChecker.Type()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to determine Checker type name for asset %s", currAssetPath)
		}
		checkerPriority, err := currChecker.Priority()
		if err != nil {
			return nil, nil, errors.Wrapf(err, "failed to determine Checker priority for asset %s", currAssetPath)
		}
		checkerTypeToAssets[checkerType] = append(checkerTypeToAssets[checkerType], currAssetPath)
		checkerCreators = append(checkerCreators, NewCreator(checkerType, checkerPriority,
			func(cfgYML []byte) (okgo.Checker, error) {
				newChecker := assetChecker{
					assetPath: currAssetPath,
					cfgYML:    string(cfgYML),
				}
				if err := newChecker.VerifyConfig(); err != nil {
					return nil, err
				}
				return &newChecker, nil
			}))
		configUpgraders = append(configUpgraders, &assetConfigUpgrader{
			typeName:  checkerType,
			assetPath: currAssetPath,
		})
	}
	var sortedKeys []okgo.CheckerType
	for k := range checkerTypeToAssets {
		sortedKeys = append(sortedKeys, k)
	}
	sort.Sort(okgo.ByCheckerType(sortedKeys))
	for _, k := range sortedKeys {
		if len(checkerTypeToAssets[k]) <= 1 {
			continue
		}
		sort.Strings(checkerTypeToAssets[k])
		return nil, nil, errors.Errorf("Checker type %s provided by multiple assets: %v", k, checkerTypeToAssets[k])
	}
	return checkerCreators, configUpgraders, nil
}

// RunCommandAndStreamOutput runs the provided exec.Cmd. The output that is generated to Stdout and Stderr for the
// command is processed in a separate goroutine. Each line is provided to the provided lineParser and the JSON
// representation of the issue returned by the parser is written to the provided stdout. This function will not return
// until the underlying command has finished executing and all of the generated output has been processed and written
// to the provided stdout.
func RunCommandAndStreamOutput(cmd *exec.Cmd, lineParser func(line string) okgo.Issue, stdout io.Writer) {
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		okgo.WriteErrorAsIssue(errors.Wrapf(err, "failed to create pipe"), stdout)
		return
	}

	cmd.Stdout = pipeW
	cmd.Stderr = pipeW

	done := make(chan bool)
	go func() {
		scanner := bufio.NewScanner(pipeR)
		for scanner.Scan() {
			issue := lineParser(scanner.Text())
			if issue == (okgo.Issue{}) {
				// skip empty issues
				continue
			}
			issueJSONBytes, err := json.Marshal(issue)
			if err != nil {
				okgo.WriteErrorAsIssue(errors.Wrapf(err, "failed to marshal issue %+v as JSON", issue), stdout)
				continue
			}
			fmt.Fprintln(stdout, string(issueJSONBytes))
		}
		if err := scanner.Err(); err != nil {
			okgo.WriteErrorAsIssue(errors.Wrapf(err, "scanner error encountered while reading output"), stdout)
		}
		done <- true
	}()

	// run command
	if err := cmd.Run(); err != nil {
		if _, ok := err.(*exec.ExitError); !ok {
			// if error is not an *exec.ExitError, record it. Do not record errors of type *exec.ExitError because it is
			// not possible to distinguish between a check that found issues and exited with a non-zero code despite
			// running successfully and a check that failed in some other manner. All execution errors must be handled
			// by writing to stdout. This does mean that a check that exits with a non-zero error code without printing
			// any output will be (incorrectly) considered as completing successfully. Such checks are not supported.
			okgo.WriteErrorAsIssue(errors.Wrapf(err, "failed to run command %v", cmd.Args), stdout)
		}
	}

	if err := pipeW.Close(); err != nil {
		<-done
		okgo.WriteErrorAsIssue(errors.Wrapf(err, "failed to close pipe writer"), stdout)
		return
	}

	// wait for all output to be processed
	<-done
}
