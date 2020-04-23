package cmd

import (
	"bytes"
	"fmt"
	"os"
	"strings"

	eirinix "github.com/SUSE/eirinix"
	"gopkg.in/yaml.v2"

	configpkg "github.com/SUSE/eirini-loggregator-bridge/config"
	. "github.com/SUSE/eirini-loggregator-bridge/logger"
	podwatcher "github.com/SUSE/eirini-loggregator-bridge/podwatcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var kubeconfig string

// See: https://github.com/spf13/viper/issues/188
// Viper, without default config doesn't set keys, and later
// they are not populated into config when unmarshalling
var config configpkg.ConfigType = configpkg.ConfigType{
	Namespace:           "default",
	LoggregatorEndpoint: "loggregator-endpoint",
	LoggregatorCAPath:   "loggregator-ca-path",
	LoggregatorCertPath: "loggregator-cert-path",
	LoggregatorKeyPath:  "loggregator-key-path",
}

var rootCmd = &cobra.Command{
	Use:   "eirini-loggregator-bridge",
	Short: "eirini-loggregator-bridge streams Eirini application logs to CloudFoundry loggregator",
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		err = config.Validate()
		if err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		filter := false
		x := eirinix.NewManager(
			eirinix.ManagerOptions{
				Namespace:           config.Namespace,
				KubeConfig:          kubeconfig,
				OperatorFingerprint: "eirini-loggregator-bridge", // Not really used for now, but setting it up for future
				FilterEiriniApps:    &filter,
			})

		x.AddWatcher(podwatcher.NewPodWatcher(config))

		err = x.Watch()
		if err != nil {
			LogError(err.Error())
			os.Exit(1)
		}
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
	rootCmd.PersistentFlags().StringVar(&kubeconfig, "kubeconfig", "", "kubeconfig file path. This is optional, in cluster config will be used if not set")
}

func initConfig() {
	if cfgFile != "" {
		viper.SetConfigType("yaml")
		emptyConfigBytes, err := yaml.Marshal(config)
		if err != nil {
			LogError("Can't marshal config:", err.Error())
			os.Exit(1)
		}

		emptyConfigReader := bytes.NewReader(emptyConfigBytes)
		viper.MergeConfig(emptyConfigReader)

		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		if err := viper.MergeInConfig(); err != nil {
			LogError("Can't read config:", err.Error())
			os.Exit(1)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))
	viper.Unmarshal(&config)

	LogDebug("Namespace: ", fmt.Sprintf("%s", config.Namespace))
	LogDebug("Loggregator-endpoint: ", fmt.Sprintf("%s", config.LoggregatorEndpoint))
	LogDebug("Loggregator-ca-path: ", fmt.Sprintf("%s", config.LoggregatorCAPath))
	LogDebug("Loggregator-cert-path: ", fmt.Sprintf("%s", config.LoggregatorCertPath))
	LogDebug("Loggregator-key-path: ", fmt.Sprintf("%s", config.LoggregatorKeyPath))
}
