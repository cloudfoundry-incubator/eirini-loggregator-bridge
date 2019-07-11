package cmd

import (
	"os"

	config "github.com/SUSE/eirini-loggregator-bridge/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string

var Config config.ConfigType

var rootCmd = &cobra.Command{
	Use:   "eirini-loggregator-bridge",
	Short: "eirini-loggregator-bridge streams Eirini application logs to CloudFoundry loggregator",
	Run: func(cmd *cobra.Command, args []string) {
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		LogError(err.Error())
		os.Exit(1)
	}
}

// Loggregator TLS:
// https://github.com/cloudfoundry/go-loggregator/blob/master/tls.go
// https://docs.cloudfoundry.org/loggregator/architecture.html
func init() {
	cobra.OnInitialize(initConfig)
	rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file")
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
	} else {
		LogError("You didn't specify a config file (Use --help)")
		os.Exit(1)
	}

	if err := viper.ReadInConfig(); err != nil {
		LogError("Can't read config:", err.Error())
		os.Exit(1)
	}
	viper.Unmarshal(&Config)
}
