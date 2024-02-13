package check

import (
	"github.com/palantir/okgo/okgo"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
	"time"
)

type inMemoryChecker struct {
	checkerType okgo.CheckerType
}

func (i inMemoryChecker) Type() (okgo.CheckerType, error) {
	return i.checkerType, nil
}

func (i inMemoryChecker) Priority() (okgo.CheckerPriority, error) {
	return 0, nil
}

func (i inMemoryChecker) Check(pkgPaths []string, projectDir string, stdout io.Writer) {
	time.Sleep(time.Second)
	// stdout.Write([]byte(`uhoh`))
}

func (i inMemoryChecker) RunCheckCmd(args []string, stdout io.Writer) {
	//TODO implement me
	panic("implement RunCheckCmd")
}

func TestRun(t *testing.T) {
	projectParam := okgo.ProjectParam{
		Checks: map[okgo.CheckerType]okgo.CheckerParam{
			"test1": {
				Skip:     false,
				Priority: nil,
				Checker:  &inMemoryChecker{checkerType: "test1"},
				Filters:  nil,
				Exclude:  nil,
			},
			"test2": {
				Skip:     false,
				Priority: nil,
				Checker:  &inMemoryChecker{checkerType: "test2"},
				Filters:  nil,
				Exclude:  nil,
			},
			"test3": {
				Skip:     false,
				Priority: nil,
				Checker:  &inMemoryChecker{checkerType: "test3"},
				Filters:  nil,
				Exclude:  nil,
			},
			"test4": {
				Skip:     false,
				Priority: nil,
				Checker:  &inMemoryChecker{checkerType: "test4"},
				Filters:  nil,
				Exclude:  nil,
			},
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
		"test2",
		"test3",
		"test4",
	}
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 2, os.Stdout)
	assert.NoError(t, err)
}
