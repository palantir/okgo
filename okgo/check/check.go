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

package check

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	"github.com/palantir/okgo/okgo"
	"github.com/pkg/errors"
)

func Run(projectParam okgo.ProjectParam, checkersToRun []okgo.CheckerType, pkgPaths []string, projectDir string, factory okgo.CheckerFactory, parallelism int, stdout io.Writer) error {
	checkers, maxTypeLen, err := getCheckersToRun(projectParam, checkersToRun, factory)
	if err != nil {
		return err
	}
	// if there are fewer checkers than max parallelism, update parallelism to number of checkers
	if len(checkers) < parallelism {
		parallelism = len(checkers)
	}

	jobs := make(chan okgo.CheckerParam, len(checkers))
	results := make(chan checkResult, len(checkers))

	var checksWithFailures []string
	pullResultsOff := func(toRun int) {
		for i := 0; i < toRun; i++ {
			checkResult := <-results
			if checkResult.producedOutput {
				checksWithFailures = append(checksWithFailures, string(checkResult.checkerType))
			}
		}
	}

	startASingleWorker := func() {
		go singleCheckWorker(pkgPaths, projectDir, maxTypeLen, parallelism > 1, jobs, results, stdout)
	}
	// Always start 1 worker no matter what
	startASingleWorker()

	// Bucket checks into serial and parallel ones and run all serial ones first
	checkersToRunSerially, checkersToRunInParallel, err := partitionCheckerJobs(checkers)
	if err != nil {
		return err
	}
	// Start all jobs that must be run serially. This enqueues all jobs in order and
	// because there is only one worker they will run one at a time.
	for _, checker := range checkersToRunSerially {
		jobs <- checker
	}
	// Retrieve all results
	pullResultsOff(len(checkersToRunSerially))

	// Finished processing serial checks: start up the rest of the workers to enable
	// maximal supported parallelism for workers
	for w := 0; w < parallelism-1; w++ {
		startASingleWorker()
	}
	// Enqueue the checks that can run in parallel
	for _, checker := range checkersToRunInParallel {
		jobs <- checker
	}
	// Retrieve the rest of the results
	pullResultsOff(len(checkersToRunInParallel))

	if len(checksWithFailures) > 0 {
		sort.Strings(checksWithFailures)
		_, _ = fmt.Fprintln(stdout, "Check(s) produced output:", checksWithFailures)
		// return empty failure to indicate non-zero exit code
		return fmt.Errorf("")
	}
	return nil
}

func getCheckersToRun(projectParam okgo.ProjectParam, checkersToRun []okgo.CheckerType, factory okgo.CheckerFactory) ([]okgo.CheckerParam, int, error) {
	var checkers []okgo.CheckerParam
	maxTypeLen := 0
	for _, checkerType := range checkersToRun {
		if len(checkerType) > maxTypeLen {
			maxTypeLen = len(checkerType)
		}
		param, ok := projectParam.Checks[checkerType]
		if ok {
			checkers = append(checkers, param)
			continue
		}
		checker, err := factory.NewChecker(checkerType, nil)
		if err != nil {
			return nil, 0, errors.Wrapf(err, "failed to create checkerType %s", checkerType)
		}
		checkers = append(checkers, okgo.CheckerParam{
			Checker: checker,
		})
	}

	// sort the checkers
	if err := sortCheckers(checkers); err != nil {
		return nil, 0, err
	}
	return checkers, maxTypeLen, nil
}

func partitionCheckerJobs(checkers []okgo.CheckerParam) ([]okgo.CheckerParam, []okgo.CheckerParam, error) {
	var (
		multiCPUCheckers  []okgo.CheckerParam
		singleCPUCheckers []okgo.CheckerParam
	)
	for _, checker := range checkers {
		multiCPU, err := checker.Checker.MultiCPU()
		if err != nil {
			return nil, nil, err
		}
		if multiCPU {
			multiCPUCheckers = append(multiCPUCheckers, checker)
		} else {
			singleCPUCheckers = append(singleCPUCheckers, checker)
		}
	}
	return multiCPUCheckers, singleCPUCheckers, nil
}

func sortCheckers(checkers []okgo.CheckerParam) error {
	var rErr error
	sort.Slice(checkers, func(i, j int) bool {
		var iPriority okgo.CheckerPriority
		if checkers[i].Priority != nil {
			iPriority = *checkers[i].Priority
		} else {
			iPriorityVal, err := checkers[i].Checker.Priority()
			if err != nil && rErr == nil {
				rErr = err
			}
			iPriority = iPriorityVal
		}

		var jPriority okgo.CheckerPriority
		if checkers[j].Priority != nil {
			jPriority = *checkers[j].Priority
		} else {
			jPriorityVal, err := checkers[j].Checker.Priority()
			if err != nil && rErr == nil {
				rErr = err
			}
			jPriority = jPriorityVal
		}

		if iPriority == jPriority {
			// if priority is the same, sort alphabetically
			iType, err := checkers[i].Checker.Type()
			if err != nil && rErr == nil {
				rErr = err
			}
			jType, err := checkers[j].Checker.Type()
			if err != nil && rErr == nil {
				rErr = err
			}
			return iType < jType
		}
		return iPriority < jPriority
	})
	if rErr != nil {
		return errors.Wrapf(rErr, "failed to determine priority or type")
	}
	return nil
}

type checkResult struct {
	checkerType    okgo.CheckerType
	producedOutput bool
}

func singleCheckWorker(pkgPaths []string, projectDir string, maxTypeLen int, multipleWorkers bool, checkJobs <-chan okgo.CheckerParam, results chan<- checkResult, stdout io.Writer) {
	for checkerParam := range checkJobs {
		results <- getCheckResultFromChecker(pkgPaths, projectDir, maxTypeLen, multipleWorkers, checkerParam, stdout)
	}
}

func getCheckResultFromChecker(
	pkgPaths []string,
	projectDir string,
	maxTypeLen int,
	multipleWorkers bool,
	checkerParam okgo.CheckerParam,
	stdout io.Writer) checkResult {
	if checkerParam.Skip {
		return checkResult{}
	}
	checkerType, err := checkerParam.Checker.Type()
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "failed to determine type for Checker: %v", err)
		return checkResult{
			checkerType:    "UNKNOWN_CHECK_TYPE",
			producedOutput: true,
		}
	}
	prefixWithPadding := ""
	if multipleWorkers {
		prefixWithPadding = fmt.Sprintf("[%s] ", checkerType) + strings.Repeat(" ", maxTypeLen-len(checkerType))
	}
	return runCheck(checkerType, prefixWithPadding, checkerParam, pkgPaths, projectDir, stdout)
}

func runCheck(checkerType okgo.CheckerType, outputPrefix string, checkerParam okgo.CheckerParam, pkgPaths []string, projectDir string, stdout io.Writer) checkResult {
	_, _ = fmt.Fprintf(stdout, "%sRunning %s...\n", outputPrefix, checkerType)

	result := checkResult{
		checkerType: checkerType,
	}
	filteredPkgPaths := getFilteredPkgPaths(checkerParam, pkgPaths)
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, "failed to create pipe")
		result.producedOutput = true
		return result
	}

	done := make(chan bool)

	go func() {
		scanner := bufio.NewScanner(pipeR)
		for scanner.Scan() {
			line := scanner.Text()
			issue := okgo.NewIssueFromJSON(line)
			if shouldSkipIssue(issue, checkerParam) {
				continue
			}
			_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, strings.Replace(issue.String(), "\n", "\n"+outputPrefix, -1))
			result.producedOutput = true
		}
		if err := scanner.Err(); err != nil {
			_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, "scanner error encountered while reading output")
			result.producedOutput = true
		}
		done <- true
	}()

	// run check
	checkerParam.Checker.Check(filteredPkgPaths, projectDir, pipeW)

	if err := pipeW.Close(); err != nil {
		<-done
		_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, "failed to close pipe writer")
		result.producedOutput = true
		return result
	}

	// wait until all output has been read
	<-done

	_, _ = fmt.Fprintf(stdout, "%sFinished %s\n", outputPrefix, checkerType)

	return result
}

func getFilteredPkgPaths(checkerParam okgo.CheckerParam, pkgPaths []string) []string {
	var filteredPkgPaths []string
	for _, pkgPath := range pkgPaths {
		if checkerParam.Exclude != nil && checkerParam.Exclude.Match(pkgPath) {
			// skip excludes
			continue
		}
		filteredPkgPaths = append(filteredPkgPaths, pkgPath)
	}
	return filteredPkgPaths
}

func shouldSkipIssue(issue okgo.Issue, checkerParam okgo.CheckerParam) bool {
	if issue.Path != "" && checkerParam.Exclude != nil && checkerParam.Exclude.Match(issue.Path) {
		// if path matches exclude, skip
		return true
	}

	// if issue matches filter, skip
	filterOut := false
	for _, filter := range checkerParam.Filters {
		if filter.Filter(issue) {
			filterOut = true
			break
		}
	}
	return filterOut
}
