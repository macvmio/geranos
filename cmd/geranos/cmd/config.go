package cmd

import (
	"fmt"
	"github.com/macvmio/geranos/pkg/appconfig"
	"github.com/spf13/viper"
)

var flagConfigFile string
var flagLocalDebug bool

var TheAppConfig appconfig.Config

func initConfig() error {
	if flagConfigFile != "" {
		viper.SetConfigFile(flagConfigFile)
	} else {
		viper.AddConfigPath("$HOME/.geranos")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}
	viper.SetEnvPrefix("GERANOS")
	viper.AutomaticEnv()

	if flagLocalDebug {
		fmt.Println("Using config file:", viper.ConfigFileUsed())
		fmt.Println(viper.AllSettings())
	}
	if err := viper.ReadInConfig(); err != nil {
		return fmt.Errorf("error reading viper config '%v': %w", viper.ConfigFileUsed(), err)
	}
	if err := viper.Unmarshal(&TheAppConfig); err != nil {
		return fmt.Errorf("error unmarshalling viper config '%v': %w", viper.ConfigFileUsed(), err)
	}
	return nil
}
