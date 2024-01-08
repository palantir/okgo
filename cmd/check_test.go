package cmd

import (
	"fmt"
	"github.com/stretchr/testify/assert"
	"os"

	"github.com/palantir/godel/v2/framework/pluginapi/v2/pluginapi"
	"testing"
)

func TestUpgradeConfig(t *testing.T) {

	err := InitAssetCmds(nil)
	assert.NoError(t, err)
	okgoConfigFileFlagVal = `/Volumes/git/go/src/github.palantir.build/deployability/instance-group-operator/godel/config/check-plugin.yml`
	godelConfigFileFlagVal = `/Volumes/git/go/src/github.palantir.build/deployability/instance-group-operator/godel/config/godel.yml`

	_, _, err = okgoProjectParamFromFlags()
	assert.NoError(t, err)
	fmt.Println("a")
}

func TestUpgradeConfig2(t *testing.T) {

	if ok := pluginapi.InfoCmd(nil, os.Stdout, PluginInfo); ok {
		return
	}
	p := []string{
		"com.palantir.godel-okgo-asset-deadcode:deadcode-asset:1.38.0",
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
