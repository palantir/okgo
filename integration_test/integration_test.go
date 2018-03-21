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

package integration_test

import (
	"testing"

	"github.com/palantir/godel/framework/pluginapitester"
	"github.com/palantir/godel/pkg/products"
	"github.com/stretchr/testify/require"
)

const (
	godelYML = `exclude:
  names:
    - "\\..+"
    - "vendor"
  paths:
    - "godel"
`
)

func TestUpgradeConfig(t *testing.T) {
	pluginPath, err := products.Bin("check-plugin")
	require.NoError(t, err)
	pluginProvider := pluginapitester.NewPluginProvider(pluginPath)

	pluginapitester.RunUpgradeConfigTest(t,
		pluginProvider,
		nil,
		[]pluginapitester.UpgradeConfigTestCase{
			{
				Name: "top-level legacy format config is upgraded properly",
				ConfigFiles: map[string]string{
					"godel/config/godel.yml": godelYML,
					"godel/config/check-plugin.yml": `
legacy-config: true
release-tag: go1.7
exclude:
  names:
    - "m?cks"
  paths:
    - "vendor"
`,
				},
				WantOutput: "Upgraded configuration for check-plugin.yml\n",
				WantFiles: map[string]string{
					"godel/config/check-plugin.yml": `release-tag: go1.7
checks: {}
exclude:
  names:
  - m?cks
  paths:
  - vendor
`,
				},
			},
		},
	)
}
