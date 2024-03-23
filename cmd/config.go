package cmd

import (
	"fmt"
	"github.com/spf13/viper"
)

var flagConfigFile string
var flagLocalDebug bool

func initConfig() {
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
	err := viper.ReadInConfig()
	if err != nil {
		fmt.Printf("error reading viper config '%v': %v", viper.ConfigFileUsed(), err)
	}
}
