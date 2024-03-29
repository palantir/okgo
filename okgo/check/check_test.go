// Copyright 2024 Palantir Technologies, Inc.
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

package check

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"os"
	"testing"
	"time"

	"github.com/palantir/okgo/okgo"
	"github.com/palantir/pkg/matcher"
	"github.com/stretchr/testify/assert"
)

type inMemoryChecker struct {
	okgo.Checker
	checkerType okgo.CheckerType
	multiCPU    okgo.CheckerMultiCPU
	issue       *okgo.Issue
	timeToWait  *time.Duration
}

func (i *inMemoryChecker) Type() (okgo.CheckerType, error) {
	if i.checkerType == "error_on_type_check" {
		return "", errors.New("test error on Type()")
	}
	return i.checkerType, nil
}

func (i *inMemoryChecker) Priority() (okgo.CheckerPriority, error) {
	return 0, nil
}

func (i *inMemoryChecker) MultiCPU() (okgo.CheckerMultiCPU, error) {
	return i.multiCPU, nil
}

func (i *inMemoryChecker) Check(pkgPaths []string, projectDir string, stdout io.Writer) {
	if i.timeToWait != nil {
		time.Sleep(*i.timeToWait)
	}
	if i.issue == nil {
		return
	}
	bytes, _ := json.Marshal(i.issue)
	_, _ = stdout.Write(bytes)
}

func TestRun_NoErrors(t *testing.T) {
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"test1": {
				Checker: &inMemoryChecker{checkerType: "test1"},
			},
			"test2": {
				Checker: &inMemoryChecker{checkerType: "test2"},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
		"test2",
	}
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, os.Stdout)
	assert.NoError(t, err)
}

func TestRun_Errors(t *testing.T) {
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"test1": {
				Checker: &inMemoryChecker{checkerType: "test1", issue: &okgo.Issue{
					Content: "output",
				}},
			},
			"test2": {
				Checker: &inMemoryChecker{checkerType: "test2"},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
		"test2",
	}
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, os.Stdout)
	assert.Error(t, err)
}

func TestRun_ErrorsButFilteredOut(t *testing.T) {
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"test1": {
				Skip: true,
				Checker: &inMemoryChecker{checkerType: "test1", issue: &okgo.Issue{
					Content: "output",
				}},
			},
			"test2": {
				Exclude: matcher.Name("p1"),
				Checker: &inMemoryChecker{checkerType: "test2", issue: &okgo.Issue{
					Path:    "p1",
					Content: "output",
				}},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
		"test2",
	}
	err := Run(projectParam, checkersToRun, []string{"p1"}, "dir", nil, 2, os.Stdout)
	assert.NoError(t, err)
}

func TestRun_ErrorsOnTypeCheck(t *testing.T) {
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"error_on_type_check": {
				Checker: &inMemoryChecker{checkerType: "error_on_type_check"},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"error_on_type_check",
	}
	buffer := &bytes.Buffer{}
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, buffer)
	assert.Error(t, err)
	assert.Contains(t, buffer.String(), "test error on Type()")
}

func TestRun_NoErrorsWithWaits(t *testing.T) {
	timeToWait := time.Millisecond * 50
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"test1": {
				Checker: &inMemoryChecker{
					checkerType: "test1",
					timeToWait:  toDuration(timeToWait),
				},
			},
			"test2": {
				Checker: &inMemoryChecker{
					checkerType: "test2",
					timeToWait:  toDuration(timeToWait),
				},
			},
			"test3": {
				Checker: &inMemoryChecker{
					checkerType: "test3",
					timeToWait:  toDuration(timeToWait),
				},
			},
			"test4": {
				Checker: &inMemoryChecker{
					checkerType: "test4",
					timeToWait:  toDuration(timeToWait),
				},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
		"test2",
		"test3",
		"test4",
	}
	// Ensure with max parallelism we can run all of them
	start := time.Now()
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 4, os.Stdout)
	assert.NoError(t, err)
	assert.Greater(t, time.Now().Sub(start), timeToWait)
	assert.Less(t, time.Now().Sub(start), timeToWait*2)

	// And scaled down we take longer
	start = time.Now()
	err = Run(projectParam, checkersToRun, nil, "dir", nil, 2, os.Stdout)
	assert.NoError(t, err)
	assert.Greater(t, time.Now().Sub(start), timeToWait*2)
	assert.Less(t, time.Now().Sub(start), timeToWait*3)
}

func TestRun_NoErrorsWithWaitsAndSplit(t *testing.T) {
	timeToWait := time.Millisecond * 50
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"test1": {
				Checker: &inMemoryChecker{
					checkerType: "test1",
					timeToWait:  toDuration(timeToWait),
					multiCPU:    true,
				},
			},
			"test2": {
				Checker: &inMemoryChecker{
					checkerType: "test2",
					timeToWait:  toDuration(timeToWait),
				},
			},
			"test3": {
				Checker: &inMemoryChecker{
					checkerType: "test3",
					timeToWait:  toDuration(timeToWait),
					multiCPU:    true,
				},
			},
			"test4": {
				Checker: &inMemoryChecker{
					checkerType: "test4",
					timeToWait:  toDuration(timeToWait),
				},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
		"test2",
		"test3",
		"test4",
	}
	start := time.Now()
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, os.Stdout)
	assert.NoError(t, err)
	assert.Greater(t, time.Now().Sub(start), timeToWait*3)
	assert.Less(t, time.Now().Sub(start), timeToWait*4)
}

func toDuration(timeToWait time.Duration) *time.Duration {
	return &timeToWait
}
