// Copyright 2018 Palantir Technologies, Inc.
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
	"go/build"
	"os/exec"
	"strings"
)

// SetGoBuildDefaultReleaseTags sets the value of go/build.Default.ReleaseTags to be the current runtime release tags
// if there are fewer release tags at runtime than at compile time.
func SetGoBuildDefaultReleaseTags() {
	runtimeGoReleaseTags := RuntimeGoReleaseTags()
	if runtimeGoReleaseTags == nil {
		return
	}

	// if the number of release tags at runtime is less than the number of release tags at compile time, the runtime
	// version of Go is older than the version of Go used to compile the check. Perform the check as if the latest
	// version is the runtime version.
	if len(runtimeGoReleaseTags) < len(build.Default.ReleaseTags) {
		build.Default.ReleaseTags = runtimeGoReleaseTags
	}
}

// RuntimeGoReleaseTags determines the Go release tags at runtime as returned by "go list -f". Returns nil if the
// runtime release tags cannot be determined.
func RuntimeGoReleaseTags() []string {
	goReleaseTagsOutput, err := exec.Command("go", "list", "-f", "{{range context.ReleaseTags}}{{println .}}{{end}}").CombinedOutput()
	if err != nil {
		return nil
	}
	return strings.Split(strings.TrimSuffix(string(goReleaseTagsOutput), "\n"), "\n")
}
