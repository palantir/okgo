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

package main

import (
	"fmt"
	"os"

	"github.com/palantir/godel/v2/framework/pluginapi/v2/pluginapi"
	"github.com/palantir/okgo/cmd"
)

func main() {
	if ok := pluginapi.InfoCmd(os.Args, os.Stdout, cmd.PluginInfo); ok {
		return
	}
	// initialize commands that require assets
	if err := cmd.InitAssetCmds(os.Args[1:]); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(cmd.Execute())
}
