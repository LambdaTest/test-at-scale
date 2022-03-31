package config

import (
	"github.com/LambdaTest/synapse/pkg/global"
	"github.com/spf13/viper"
)

func setNucleusDefaultConfig() {
	viper.SetDefault("LogConfig.EnableConsole", true)
	viper.SetDefault("LogConfig.ConsoleJSONFormat", false)
	viper.SetDefault("LogConfig.ConsoleLevel", "debug")
	viper.SetDefault("LogConfig.EnableFile", true)
	viper.SetDefault("LogConfig.FileJSONFormat", true)
	viper.SetDefault("LogConfig.FileLevel", "debug")
	viper.SetDefault("LogConfig.FileLocation", global.HomeDir+"/nucleus.log")
	viper.SetDefault("Env", "prod")
	viper.SetDefault("Port", "9876")
	viper.SetDefault("Verbose", false)
}

func setSynapseDefaultConfig() {
	viper.SetDefault("LogConfig.EnableConsole", true)
	viper.SetDefault("LogConfig.ConsoleJSONFormat", false)
	viper.SetDefault("LogConfig.ConsoleLevel", "info")
	viper.SetDefault("LogConfig.EnableFile", true)
	viper.SetDefault("LogConfig.FileJSONFormat", true)
	viper.SetDefault("LogConfig.FileLevel", "debug")
	viper.SetDefault("LogConfig.FileLocation", "./mould.log")
	viper.SetDefault("Env", "prod")
	viper.SetDefault("Verbose", false)
}
