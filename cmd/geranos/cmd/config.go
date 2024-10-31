package cmd

import (
	"fmt"
	"github.com/macvmio/geranos/pkg/appconfig"
	"github.com/spf13/viper"
	"os"
	"path"
)

var flagConfigFile string
var flagLocalDebug bool

var TheAppConfig appconfig.Config

func initConfig() error {
	if flagConfigFile != "" {
		viper.SetConfigFile(flagConfigFile)
	} else {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not determine home directory: %w", err)
		}
		viper.AddConfigPath(path.Join(home, ".geranos"))
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
