package cmd

import (
	"bytes"
	"io/ioutil"
	"os"

	eirinix "github.com/SUSE/eirinix"

	configpkg "github.com/SUSE/eirini-loggregator-bridge/config"
	. "github.com/SUSE/eirini-loggregator-bridge/logger"
	podwatcher "github.com/SUSE/eirini-loggregator-bridge/podwatcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var kubeconfig string

var config configpkg.ConfigType

var rootCmd = &cobra.Command{
	Use:   "eirini-loggregator-bridge",
	Short: "eirini-loggregator-bridge streams Eirini application logs to CloudFoundry loggregator",
	Run: func(cmd *cobra.Command, args []string) {
		var err error

		LogDebug("Namespace: ", config.Namespace)
		LogDebug("Loggregator-endpoint: ", config.LoggregatorEndpoint)
		LogDebug("Loggregator-ca-path: ", config.LoggregatorCAPath)
		LogDebug("Loggregator-cert-path: ", config.LoggregatorCertPath)
		LogDebug("Loggregator-key-path: ", config.LoggregatorKeyPath)
		LogDebug("Starting Loggregator")

		err = config.Validate()
		if err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		filter := false

		x := eirinix.NewManager(eirinix.ManagerOptions{
			Namespace:           config.Namespace,
			KubeConfig:          kubeconfig,
			OperatorFingerprint: "eirini-loggregator-bridge", // Not really used for now, but setting it up for future
			FilterEiriniApps:    &filter,
		})

		pw := podwatcher.NewPodWatcher(config)
		// Setup does need the manager to get kubernetes connection
		if err := pw.EnsureLogStream(x); err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		x.AddWatcher(pw)

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

	// As Viper cannot unmarshal and merge configs from yaml automatically,
	// define inline there the mapping explictly.
	// See: https://github.com/spf13/viper/issues/761
	viper.SetDefault("NAMESPACE", "")
	viper.SetDefault("LOGGREGATOR_KEY_PATH", "")
	viper.SetDefault("LOGGREGATOR_ENDPOINT", "")
	viper.SetDefault("LOGGREGATOR_CA_PATH", "")
	viper.SetDefault("LOGGREGATOR_CERT_PATH", "")
	viper.BindEnv("namespace", "NAMESPACE")
	viper.BindEnv("loggregator-key-path", "LOGGREGATOR_KEY_PATH")
	viper.BindEnv("loggregator-endpoint", "LOGGREGATOR_ENDPOINT")
	viper.BindEnv("loggregator-ca-path", "LOGGREGATOR_CA_PATH")
	viper.BindEnv("loggregator-cert-path", "LOGGREGATOR_CERT_PATH")

	if cfgFile != "" {
		yamlFile, err := ioutil.ReadFile(cfgFile)
		if err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		viper.SetConfigType("yaml")
		viper.ReadConfig(bytes.NewBuffer(yamlFile))
	}

	// Now this call will take into account the env as well
	err := viper.Unmarshal(&config)
	if err != nil {
		LogError(err.Error())
		os.Exit(1)
	}
}
