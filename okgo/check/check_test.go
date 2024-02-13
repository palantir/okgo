package check

import (
	"github.com/palantir/okgo/okgo"
	"github.com/stretchr/testify/assert"
	"io"
	"os"
	"testing"
)

type inMemoryChecker struct {
	checkerType okgo.CheckerType
}

func (i inMemoryChecker) Type() (okgo.CheckerType, error) {
	return i.checkerType, nil
}

func (i inMemoryChecker) Priority() (okgo.CheckerPriority, error) {
	//TODO implement me
	panic("implement Priority")
}

func (i inMemoryChecker) Check(pkgPaths []string, projectDir string, stdout io.Writer) {
	stdout.Write([]byte(`uhoh`))
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
		},
	}
	checkersToRun := []okgo.CheckerType{
		"test1",
	}
	err := Run(projectParam, checkersToRun, nil, "dir", nil, 1, os.Stdout)
	assert.NoError(t, err)
}
