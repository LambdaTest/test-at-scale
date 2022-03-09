package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"strings"

	"github.com/LambdaTest/synapse/pkg/lumber"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// GlobalNucleusConfig stores the config instance for global use
var GlobalNucleusConfig *NucleusConfig

// GlobalSynapseConfig store the config instance for synapse global use
var GlobalSynapseConfig *SynapseConfig

type tempSecretReader struct {
	RepoSecrets map[string]map[string]string `json:"RepoSecrets" yaml:"RepoSecrets"`
}

// LoadNucleusConfig loads config from command instance to predefined config variables
func LoadNucleusConfig(cmd *cobra.Command) (*NucleusConfig, error) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	// default viper configs
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// set default configs
	setNucleusDefaultConfig()

	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".nucleus")
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.nucleus")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Warning: No configuration file found. Proceeding with defaults")
	}

	return populateNucleusConfig(new(NucleusConfig))
}

// LoadSynapseConfig loads config from command instance to predefined config variables
func LoadSynapseConfig(cmd *cobra.Command) (*SynapseConfig, error) {
	err := viper.BindPFlags(cmd.Flags())
	if err != nil {
		return nil, err
	}

	// default viper configs
	viper.SetEnvPrefix("SYN")
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.AutomaticEnv()

	// set default configs
	setSynapseDefaultConfig()

	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".synapse")
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.synapse")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Warning: No configuration file found. Proceeding with defaults")
	}
	return populateSynapseConfig(new(SynapseConfig))
}

// LoadRepoSecrets loads repo secrets from configuration file
func LoadRepoSecrets(cmd *cobra.Command, synapseConfig *SynapseConfig) error {
	if configFile, _ := cmd.Flags().GetString("config"); configFile != "" {
		viper.SetConfigFile(configFile)
	} else {
		viper.SetConfigName(".synapse")
		viper.AddConfigPath("./")
		viper.AddConfigPath("$HOME/.synapse")
	}

	if err := viper.ReadInConfig(); err != nil {
		fmt.Println("Warning: No configuration file found. Proceeding with defaults")
	}

	secretFile, err := ioutil.ReadFile(viper.GetViper().ConfigFileUsed())
	if err != nil {
		fmt.Printf("error in reading config file: %v\n", err)
	}

	var tempSecret tempSecretReader
	if err := json.Unmarshal(secretFile, &tempSecret); err != nil {
		fmt.Printf("error in umarshaling secrets: %v\n", err)
	}

	synapseConfig.RepoSecrets = tempSecret.RepoSecrets
	return nil
}

// ValidateCfg checks the validity of the config
func ValidateCfg(cfg *SynapseConfig, logger lumber.Logger) error {
	if cfg.Lambdatest.SecretKey == "" {
		return errors.New("error finding lambdatest secretkey in configuration file")
	}
	if cfg.ContainerRegistry.Mode == "" {
		return errors.New("error finding ContainerRegistry Mode in configuration file")
	}
	if cfg.RepoSecrets == nil {
		logger.Debugf("no RepoSecrets found in configuration file.")
		return nil
	}
	return nil
}
