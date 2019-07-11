package cmd

import (
	"os"

	configpkg "github.com/SUSE/eirini-loggregator-bridge/config"
	podwatcher "github.com/SUSE/eirini-loggregator-bridge/podwatcher"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"k8s.io/client-go/kubernetes"
	clientcmd "k8s.io/client-go/tools/clientcmd"
)

var cfgFile string

var config configpkg.ConfigType
var kubeClient kubernetes.Clientset

var rootCmd = &cobra.Command{
	Use:   "eirini-loggregator-bridge",
	Short: "eirini-loggregator-bridge streams Eirini application logs to CloudFoundry loggregator",
	Run: func(cmd *cobra.Command, args []string) {
		kubeClient, err := GetKubeClient()
		if err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		podwatcher.NewPodWatcher(config, kubeClient).Run()
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
	viper.Unmarshal(&config)
}

// Returns a kube client to be used to talk to the kube API
// For now only works in cluster.
func GetKubeClient() (*kubernetes.Clientset, error) {
	// InClusterConfig when flags are empty
	c, err := clientcmd.BuildConfigFromFlags("", "")
	if err != nil {
		return nil, errors.Wrap(err, "failed to get inClusterConfig")
	}

	clientset, err := kubernetes.NewForConfig(c)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create clientset")
	}

	return clientset, nil
}
