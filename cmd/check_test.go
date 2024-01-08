package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"

	"github.com/palantir/godel/v2/framework/pluginapi/v2/pluginapi"
	"testing"
)

func TestUpgradeConfig(t *testing.T) {
	okgoConfigFileFlagVal = `/Volumes/git/go/src/github.palantir.build/deployability/instance-group-operator/godel/config/check-plugin.yml`
	godelConfigFileFlagVal = `/Volumes/git/go/src/github.palantir.build/deployability/instance-group-operator/godel/config/godel.yml`
	projectDirFlagVal = `/Volumes/git/go/src/github.palantir.build/deployability/instance-group-operator`
	p := []string{
		"/Users/ksimons/.godel/assets/com.palantir.godel-okgo-asset-deadcode-deadcode-asset-1.38.0",
		"/Users/ksimons/.godel/assets/com.palantir.godel-okgo-asset-golint-golint-asset-1.30.0",
		"/Users/ksimons/.godel/assets/com.palantir.godel-okgo-asset-importalias-importalias-asset-1.33.0",
		"/Users/ksimons/.godel/assets/com.palantir.godel-okgo-asset-varcheck-varcheck-asset-1.38.0",
		"/Users/ksimons/.godel/assets/com.palantir.godel-okgo-asset-compiles-compiles-asset-1.41.0",
	}
	assetsFlagVal = p
	err := InitAssetCmds(nil)
	assert.NoError(t, err)

	err = runMe()
	assert.NoError(t, err)

}

func TestUpgradeConfig2(t *testing.T) {

	if ok := pluginapi.InfoCmd(nil, os.Stdout, PluginInfo); ok {
		return
	}
	p := []string{
		"Users/ksimons/.godel/assets/com.palantir.godel-okgo-asset-deadcode-deadcode-asset-1.38.0",
	}
	assetsFlagVal = p
	// initialize commands that require assets
	if err := InitAssetCmds(p); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
	os.Exit(Execute())

	err := runMe()
	assert.NoError(t, err)
	fmt.Println("a")
}
