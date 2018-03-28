package app

import (
	"testing"

	"github.com/spf13/viper"
)

const testCfgFile = "../config/test.yaml"
const invalidCfgFile = "../config/invalid"

func TestGetApp(t *testing.T) {
	viper.SetDefault("logger.level", "DEBUG")
	viper.SetConfigFile(testCfgFile)
	app := GetApp("127.0.0.1", 9999, false, testCfgFile)
	if app.Port != 9999 || app.Host != "127.0.0.1" {
		t.Fail()
	}

	defer func() {
		if r := recover(); r == nil {
			t.Fail()
		}
	}()

	viper.SetConfigFile(invalidCfgFile)
	app = GetApp("127.0.0.1", 9999, false, invalidCfgFile)
}
