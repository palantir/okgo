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

package okgotester

import (
	"bytes"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
	"testing"

	"github.com/nmiyake/pkg/dirs"
	"github.com/nmiyake/pkg/gofiles"
	"github.com/palantir/godel/framework/artifactresolver"
	"github.com/palantir/godel/framework/pluginapitester"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type AssetTestCase struct {
	Name        string
	Specs       []gofiles.GoFileSpec
	ConfigFiles map[string]string
	Wd          string
	WantError   bool
	WantOutput  string
}

// RunAssetCheckTest tests the "check" operation using the provided asset. Resolves the okgo plugin using the provided
// locator and resolver, provides it with the asset and invokes the "check" command for the specified asset.
func RunAssetCheckTest(t *testing.T,
	okgoPluginLocator, okgoPluginResolver string,
	assetPath, checkName string,
	testCases []AssetTestCase,
) {

	tmpDir, cleanup, err := dirs.TempDir("", "")
	require.NoError(t, err)
	defer cleanup()

	tmpDir, err = filepath.EvalSymlinks(tmpDir)
	require.NoError(t, err)

	for i, tc := range testCases {
		projectDir, err := ioutil.TempDir(tmpDir, "")
		require.NoError(t, err)

		var sortedKeys []string
		for k := range tc.ConfigFiles {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		for _, k := range sortedKeys {
			err = os.MkdirAll(path.Dir(path.Join(projectDir, k)), 0755)
			require.NoError(t, err)
			err = ioutil.WriteFile(path.Join(projectDir, k), []byte(tc.ConfigFiles[k]), 0644)
			require.NoError(t, err)
		}

		_, err = gofiles.Write(projectDir, tc.Specs)
		require.NoError(t, err)

		outputBuf := &bytes.Buffer{}
		pluginCfg := artifactresolver.LocatorWithResolverConfig{
			Locator: artifactresolver.LocatorConfig{
				ID: okgoPluginLocator,
			},
			Resolver: okgoPluginResolver,
		}
		pluginsParam, err := pluginCfg.ToParam()
		require.NoError(t, err)

		func() {
			wd, err := os.Getwd()
			require.NoError(t, err)

			wantWd := projectDir
			if tc.Wd != "" {
				wantWd = path.Join(wantWd, tc.Wd)
			}
			err = os.Chdir(wantWd)
			require.NoError(t, err)
			defer func() {
				err = os.Chdir(wd)
				require.NoError(t, err)
			}()

			runPluginCleanup, err := pluginapitester.RunAsset(pluginsParam, []string{assetPath}, "check", []string{
				checkName,
			}, projectDir, false, outputBuf)
			defer runPluginCleanup()
			if tc.WantError {
				require.EqualError(t, err, "", "Case %d: %s", i, tc.Name)
			} else {
				require.NoError(t, err, "Case %d: %s", i, tc.Name)
			}
			assert.Equal(t, tc.WantOutput, outputBuf.String(), "Case %d: %s", i, tc.Name)
		}()
	}
}
