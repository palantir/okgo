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
	"sync"
	"time"

	"github.com/palantir/okgo/okgo"
	"github.com/pkg/errors"
)

func Run(projectParam okgo.ProjectParam, checkersToRun []okgo.CheckerType, pkgPaths []string, projectDir string, factory okgo.CheckerFactory, parallelism int, stdout io.Writer) error {
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
			return errors.Wrapf(err, "failed to create checkerType %s", checkerType)
		}
		checkers = append(checkers, okgo.CheckerParam{
			Checker: checker,
		})
	}

	// sort the checkers
	if err := sortCheckers(checkers); err != nil {
		return err
	}
	return runNew(checkers, pkgPaths, projectDir, parallelism)
}

func runNew(
	checkers []okgo.CheckerParam,
	pkgPaths []string,
	projectDir string,
	parallelism int) error {

	var wg sync.WaitGroup

	fmt.Println("RUN NEW")
	for _, checker := range checkers {
		wg.Add(1)
		toRun := checker
		go func() {
			defer wg.Done()
			oneRun(pkgPaths, projectDir, toRun)
		}()
	}
	wg.Wait()
	return nil
}

func runOld(
	checkers []okgo.CheckerParam,
	pkgPaths []string,
	projectDir string,
	parallelism int) error {
	jobs := make(chan okgo.CheckerParam)
	results := make(chan checkResult, len(checkers))

	// if there are fewer checkers than max parallelism, update parallelism to number of checkers
	if len(checkers) < parallelism {
		parallelism = len(checkers)
	}

	for w := 0; w < parallelism; w++ {
		go singleCheckWorker(pkgPaths, projectDir, 10, parallelism > 1, jobs, results, os.Stdout)
	}

	for _, checker := range checkers {
		jobs <- checker
	}
	close(jobs)

	var checksWithFailures []string
	for range checkers {
		checkResult := <-results
		if checkResult.producedOutput {
			checksWithFailures = append(checksWithFailures, string(checkResult.checkerType))
		}
	}
	if len(checksWithFailures) > 0 {
		sort.Strings(checksWithFailures)
		_, _ = fmt.Fprintln(os.Stdout, "Check(s) produced output:", checksWithFailures)
		// return empty failure to indicate non-zero exit code
		return fmt.Errorf("")
	}
	return nil
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
		if checkerParam.Skip {
			results <- checkResult{}
			continue
		}

		checkerType, err := checkerParam.Checker.Type()
		if err != nil {
			_, _ = fmt.Fprintf(stdout, "failed to determine type for Checker: %v", err)
			continue
		}
		prefixWithPadding := ""
		if multipleWorkers {
			prefixWithPadding = fmt.Sprintf("[%s] ", checkerType) + strings.Repeat(" ", maxTypeLen-len(checkerType))
		}
		results <- runCheck3(checkerType, prefixWithPadding, checkerParam, pkgPaths, projectDir, stdout)
	}
}

func oneRun(
	pkgPaths []string,
	projectDir string,
	checkerParam okgo.CheckerParam) {
	checkerType, _ := checkerParam.Checker.Type()
	var filteredPkgPaths []string
	for _, pkgPath := range pkgPaths {
		if checkerParam.Exclude != nil && checkerParam.Exclude.Match(pkgPath) {
			// skip excludes
			continue
		}
		filteredPkgPaths = append(filteredPkgPaths, pkgPath)
	}
	fmt.Println(fmt.Sprintf("vMIDDLE: %s %s", checkerType, time.Now().String()))
	filteredPkgPaths = []string{"./cmd"}
	fmt.Println(filteredPkgPaths)
	checkerParam.Checker.Check(filteredPkgPaths, projectDir, os.Stdout)
	fmt.Println(fmt.Sprintf("vMIDDLE_2: %s %s", checkerType, time.Now().String()))
}

func runCheck3(checkerType okgo.CheckerType, outputPrefix string, checkerParam okgo.CheckerParam, pkgPaths []string, projectDir string, stdout io.Writer) checkResult {
	fmt.Println(fmt.Sprintf("vSTART: %s %s", checkerType, time.Now().String()))
	_, _ = fmt.Fprintf(stdout, "%sRunning %s...\n", outputPrefix, checkerType)

	result := checkResult{
		checkerType: checkerType,
	}
	var filteredPkgPaths []string
	for _, pkgPath := range pkgPaths {
		if checkerParam.Exclude != nil && checkerParam.Exclude.Match(pkgPath) {
			// skip excludes
			continue
		}
		filteredPkgPaths = append(filteredPkgPaths, pkgPath)
	}

	_, pipeW, err := os.Pipe()
	if err != nil {
		_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, "failed to create pipe")
		result.producedOutput = true
		return result
	}

	// run check
	fmt.Println(fmt.Sprintf("vMIDDLE: %s %s", checkerType, time.Now().String()))
	checkerParam.Checker.Check(filteredPkgPaths, projectDir, pipeW)
	fmt.Println(fmt.Sprintf("vMIDDLE_2: %s %s", checkerType, time.Now().String()))

	if err := pipeW.Close(); err != nil {
		_, _ = fmt.Fprintf(stdout, "%s%s\n", outputPrefix, "failed to close pipe writer")
		result.producedOutput = true
		return result
	}

	_, _ = fmt.Fprintf(stdout, "%sFinished %s\n", outputPrefix, checkerType)
	fmt.Println(fmt.Sprintf("DONE: %s %s", checkerType, time.Now().String()))
	return result
}

func runCheckOld(checkerType okgo.CheckerType, outputPrefix string, checkerParam okgo.CheckerParam, pkgPaths []string, projectDir string, stdout io.Writer) checkResult {
	fmt.Println(fmt.Sprintf("START: %s %s", checkerType, time.Now().String()))
	_, _ = fmt.Fprintf(stdout, "%sRunning %s...\n", outputPrefix, checkerType)

	result := checkResult{
		checkerType: checkerType,
	}
	var filteredPkgPaths []string
	for _, pkgPath := range pkgPaths {
		if checkerParam.Exclude != nil && checkerParam.Exclude.Match(pkgPath) {
			// skip excludes
			continue
		}
		filteredPkgPaths = append(filteredPkgPaths, pkgPath)
	}

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

			if issue.Path != "" && checkerParam.Exclude != nil && checkerParam.Exclude.Match(issue.Path) {
				// if path matches exclude, skip
				continue
			}

			// if issue matches filter, skip
			filterOut := false
			for _, filter := range checkerParam.Filters {
				if filter.Filter(issue) {
					filterOut = true
					break
				}
			}
			if filterOut {
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
	fmt.Println(fmt.Sprintf("DONE: %s %s", checkerType, time.Now().String()))
	return result
}
