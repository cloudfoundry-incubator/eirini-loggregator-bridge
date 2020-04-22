package cmd

import (
	"fmt"
	"os"
	"reflect"
	"strings"

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

// See: https://github.com/spf13/viper/issues/188#issuecomment-399884438
func BindEnvs(iface interface{}, parts ...string) {
	ifv := reflect.ValueOf(iface)
	ift := reflect.TypeOf(iface)
	for i := 0; i < ift.NumField(); i++ {
		v := ifv.Field(i)
		t := ift.Field(i)
		tv, ok := t.Tag.Lookup("mapstructure")
		if !ok {
			continue
		}
		switch v.Kind() {
		case reflect.Struct:
			BindEnvs(v.Interface(), append(parts, tv)...)
		default:
			viper.BindEnv(strings.Join(append(parts, tv), "."))
		}
	}
}

func initConfig() {
	if cfgFile != "" {
		// Use config file from the flag.
		viper.SetConfigFile(cfgFile)
		if err := viper.ReadInConfig(); err != nil {
			LogError("Can't read config:", err.Error())
			os.Exit(1)
		}
	}

	viper.AutomaticEnv() // read in environment variables that match
	viper.SetEnvKeyReplacer(strings.NewReplacer("-", "_"))

	BindEnvs(config)
	viper.Unmarshal(&config)

	LogDebug("Namespace: ", fmt.Sprintf("%s", config.Namespace))
	LogDebug("Loggregator-endpoint: ", fmt.Sprintf("%s", config.LoggregatorEndpoint))
	LogDebug("Loggregator-ca-path: ", fmt.Sprintf("%s", config.LoggregatorCAPath))
	LogDebug("Loggregator-cert-path: ", fmt.Sprintf("%s", config.LoggregatorCertPath))
	LogDebug("Loggregator-key-path: ", fmt.Sprintf("%s", config.LoggregatorKeyPath))
}
