package check

import (
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
	checkerType okgo.CheckerType
	multiCPU    okgo.CheckerMultiCPU
	issue       *okgo.Issue
	times       int
	timeToWait  *time.Duration
}

func (i *inMemoryChecker) Type() (okgo.CheckerType, error) {
	if i.checkerType == "error_check" {
		return "", errors.New("hi")
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

func (i *inMemoryChecker) RunCheckCmd(args []string, stdout io.Writer) {
	panic("implement RunCheckCmd")
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
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, NewDebugLogger(true), os.Stdout)
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
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, NewDebugLogger(true), os.Stdout)
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
	err := Run(projectParam, checkersToRun, []string{"p1"}, "dir", nil, 2, NewDebugLogger(true), os.Stdout)

	assert.NoError(t, err)
}

func TestRun_Hang(t *testing.T) {
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"error_check": {
				Checker: &inMemoryChecker{checkerType: "error_check", issue: &okgo.Issue{
					Content: "output",
				}},
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"error_check",
	}
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, NewDebugLogger(true), os.Stdout)
	assert.Error(t, err)
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
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 4, NewDebugLogger(true), os.Stdout)
	assert.NoError(t, err)
	assert.Greater(t, time.Now().Sub(start), timeToWait)
	assert.Less(t, time.Now().Sub(start), timeToWait*2)

	// And scaled down we take longer
	start = time.Now()
	err = Run(projectParam, checkersToRun, nil, "dir", nil, 2, NewDebugLogger(true), os.Stdout)
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
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, NewDebugLogger(true), os.Stdout)
	assert.NoError(t, err)
	assert.Greater(t, time.Now().Sub(start), timeToWait*3)
	assert.Less(t, time.Now().Sub(start), timeToWait*4)
}

func toDuration(timeToWait time.Duration) *time.Duration {
	return &timeToWait
}
