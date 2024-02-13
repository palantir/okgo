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

func Run(
	projectParam okgo.ProjectParam,
	checkersToRun []okgo.CheckerType,
	pkgPaths []string,
	projectDir string,
	factory okgo.CheckerFactory,
	parallelism int,
	debugLogger DebugLogger,
	stdout io.Writer) error {
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
		go singleCheckWorker(pkgPaths, projectDir, maxTypeLen, parallelism > 1, jobs, results, debugLogger, stdout)
	}
	// We always have 1 worker no matter what
	startASingleWorker()

	// First run all jobs that need to be ran one at a time
	var checkersWeCannotRunInParallel []okgo.CheckerParam
	var checkersWeCanRunInParallel []okgo.CheckerParam
	for _, checker := range checkers {
		if doesCheckerSupportParallelism(checker) {
			checkersWeCannotRunInParallel = append(checkersWeCannotRunInParallel, checker)
		} else {
			checkersWeCanRunInParallel = append(checkersWeCanRunInParallel, checker)
		}
	}
	// Start all the single ones
	for _, checker := range checkersWeCannotRunInParallel {
		jobs <- checker
	}
	// And then pull off those results
	pullResultsOff(len(checkersWeCannotRunInParallel))

	// Now we can queue up the rest
	for _, checker := range checkersWeCanRunInParallel {
		jobs <- checker
	}

	// And start more workers, minus the one that is running
	for w := 0; w < parallelism-1; w++ {
		startASingleWorker()
	}

	// And then we pull off the rest
	pullResultsOff(len(checkersWeCanRunInParallel))

	if len(checksWithFailures) > 0 {
		sort.Strings(checksWithFailures)
		_, _ = fmt.Fprintln(stdout, "Check(s) produced output:", checksWithFailures)
		// return empty failure to indicate non-zero exit code
		return fmt.Errorf("")
	}
	return nil
}

func doesCheckerSupportParallelism(checkerParam okgo.CheckerParam) bool {
	checkerT, _ := checkerParam.Checker.Type()
	checkerType := string(checkerT)
	return strings.Contains(checkerType, "errcheck") || strings.Contains(checkerType, "compiles")
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

func singleCheckWorker(
	pkgPaths []string,
	projectDir string,
	maxTypeLen int,
	multipleWorkers bool,
	checkJobs <-chan okgo.CheckerParam,
	results chan<- checkResult,
	debugLogger DebugLogger,
	stdout io.Writer) {
	for checkerParam := range checkJobs {
		results <- getCheckResultForCheck(pkgPaths, projectDir, maxTypeLen, multipleWorkers, checkerParam, debugLogger, stdout)
	}
}

func getCheckResultForCheck(
	pkgPaths []string,
	projectDir string,
	maxTypeLen int,
	multipleWorkers bool,
	checkerParam okgo.CheckerParam,
	debugLogger DebugLogger,
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
	return runCheck(checkerType, prefixWithPadding, checkerParam, pkgPaths, projectDir, debugLogger, stdout)
}

func runCheck(
	checkerType okgo.CheckerType,
	outputPrefix string,
	checkerParam okgo.CheckerParam,
	pkgPaths []string,
	projectDir string,
	debugLogger DebugLogger,
	stdout io.Writer) checkResult {
	result := checkResult{
		checkerType: checkerType,
	}
	producedOutput, err := runCheckAndPrintOutput(checkerType, outputPrefix, checkerParam, pkgPaths, projectDir, debugLogger, stdout)
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, err.Error())
		result.producedOutput = true
	} else {
		result.producedOutput = producedOutput
	}
	return result
}

func runCheckAndPrintOutput(
	checkerType okgo.CheckerType,
	outputPrefix string,
	checkerParam okgo.CheckerParam,
	pkgPaths []string,
	projectDir string,
	debugLogger DebugLogger,
	stdout io.Writer) (bool, error) {
	filteredPkgPaths := getFilteredPkgPaths(checkerParam, pkgPaths)
	_, _ = fmt.Fprintf(stdout, "%sRunning %s...\n", outputPrefix, checkerType)
	pipeR, pipeW, err := os.Pipe()
	if err != nil {
		return false, err
	}
	// run check
	debugLogger.Log(fmt.Sprintf("Start running check for %s", checkerType))
	checkerParam.Checker.Check(filteredPkgPaths, projectDir, pipeW)
	debugLogger.Log(fmt.Sprintf("Finished running check for %s", checkerType))

	// Close the pipe in which the output was written to
	if err := pipeW.Close(); err != nil {
		return false, err
	}
	producedOutput := false

	// Pull that through the filtering process, we will continue on errors to ensure all output is printed
	scanner := bufio.NewScanner(pipeR)
	for scanner.Scan() {
		line := scanner.Text()
		issue := okgo.NewIssueFromJSON(line)
		if shouldSkipIssue(issue, checkerParam) {
			continue
		}
		_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, strings.Replace(issue.String(), "\n", "\n"+outputPrefix, -1))
		producedOutput = true
	}

	if err := scanner.Err(); err != nil {
		return false, err
	}
	_, _ = fmt.Fprintf(stdout, "%sFinished %s\n", outputPrefix, checkerType)
	return producedOutput, nil
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
