package cmd

import (
	"bytes"
	"context"
	"io/ioutil"
	"os"

	eirinix "code.cloudfoundry.org/eirinix"

	configpkg "code.cloudfoundry.org/eirini-loggregator-bridge/config"
	. "code.cloudfoundry.org/eirini-loggregator-bridge/logger"
	podwatcher "code.cloudfoundry.org/eirini-loggregator-bridge/podwatcher"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var cfgFile string
var kubeconfig string

var config configpkg.ConfigType

var rootCmd = &cobra.Command{
	Use:   "eirini-loggregator-bridge",
	Short: "eirini-loggregator-bridge streams Eirini application logs to CloudFoundry loggregator",
	PreRun: func(cmd *cobra.Command, args []string) {

		viper.BindPFlag("operator-webhook-host", cmd.Flags().Lookup("operator-webhook-host"))
		viper.BindPFlag("operator-webhook-port", cmd.Flags().Lookup("operator-webhook-port"))
		viper.BindPFlag("operator-service-name", cmd.Flags().Lookup("operator-service-name"))
		viper.BindPFlag("operator-webhook-namespace", cmd.Flags().Lookup("operator-webhook-namespace"))
		viper.BindPFlag("register", cmd.Flags().Lookup("register"))
		viper.BindPFlag("graceful-fail-time", cmd.Flags().Lookup("graceful-fail-time"))
		viper.BindPFlag("graceful-success-time", cmd.Flags().Lookup("graceful-success-time"))
		viper.BindPFlag("downloader-entrypoint", cmd.Flags().Lookup("downloader-entrypoint"))
		viper.BindPFlag("executor-entrypoint", cmd.Flags().Lookup("executor-entrypoint"))
		viper.BindPFlag("uploader-entrypoint", cmd.Flags().Lookup("uploader-entrypoint"))
		viper.BindPFlag("opi-entrypoint", cmd.Flags().Lookup("opi-entrypoint"))

	},
	Run: func(cmd *cobra.Command, args []string) {
		var err error
		webhookHost := viper.GetString("operator-webhook-host")
		webhookPort := viper.GetInt32("operator-webhook-port")
		serviceName := viper.GetString("operator-service-name")
		webhookNamespace := viper.GetString("operator-webhook-namespace")
		register := viper.GetBool("register")
		gracefulFailTime := viper.GetString("graceful-fail-time")
		gracefulSuccessTime := viper.GetString("graceful-success-time")
		downloaderEntrypoint := viper.GetString("downloader-entrypoint")
		executorEntrypoint := viper.GetString("executor-entrypoint")
		uploaderEntrypoint := viper.GetString("uploader-entrypoint")
		opiEntrypoint := viper.GetString("opi-entrypoint")

		LogDebug("Namespace: ", config.Namespace)
		LogDebug("Loggregator-endpoint: ", config.LoggregatorEndpoint)
		LogDebug("Loggregator-ca-path: ", config.LoggregatorCAPath)
		LogDebug("Loggregator-cert-path: ", config.LoggregatorCertPath)
		LogDebug("Loggregator-key-path: ", config.LoggregatorKeyPath)

		LogDebug("Webhook listening on: ", webhookHost, webhookPort)
		LogDebug("Webhook namespace: ", webhookNamespace)
		LogDebug("Webhook serviceName: ", serviceName)
		LogDebug("Webhook register: ", register)

		LogDebug("Starting Loggregator")
		if webhookHost == "" {
			LogDebug("required flag 'operator-webhook-host' not set (env variable: OPERATOR_WEBHOOK_HOST)")
		}

		RegisterWebhooks := true
		if !register {
			LogDebug("The extension will start without registering")
			RegisterWebhooks = false
		}
		err = config.Validate()
		if err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		filter := false
		ctx := context.Background()
		x := eirinix.NewManager(eirinix.ManagerOptions{
			Namespace:           config.Namespace,
			KubeConfig:          kubeconfig,
			Context:             &ctx,
			OperatorFingerprint: "eirini-loggregator-bridge", // Not really used for now, but setting it up for future
			FilterEiriniApps:    &filter,

			Host:             webhookHost,
			Port:             webhookPort,
			ServiceName:      serviceName,
			WebhookNamespace: webhookNamespace,
			RegisterWebHook:  &RegisterWebhooks,
		})

		pw := podwatcher.NewPodWatcher(config)
		// Setup does need the manager to get kubernetes connection
		if err := pw.EnsureLogStream(ctx, x); err != nil {
			LogError(err.Error())
			os.Exit(1)
		}

		x.AddExtension(podwatcher.NewGracePeriodInjector(&podwatcher.GraceOptions{
			FailGracePeriod:    gracefulFailTime,
			SuccessGracePeriod: gracefulSuccessTime,

			StagingDownloaderEntrypoint: downloaderEntrypoint,
			StagingExecutorEntrypoint:   executorEntrypoint,
			StagingUploaderEntrypoint:   uploaderEntrypoint,
			RuntimeEntrypoint:           opiEntrypoint,
		}))
		x.AddExtension(pw)

		if err = x.Start(); err != nil {
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
	rootCmd.PersistentFlags().StringP("operator-webhook-host", "w", "", "Hostname/IP under which the webhook server can be reached from the cluster")
	rootCmd.PersistentFlags().StringP("operator-webhook-port", "p", "2999", "Port the webhook server listens on")
	rootCmd.PersistentFlags().StringP("operator-service-name", "s", "", "Service name where the webhook runs on (Optional, only needed inside kube)")
	rootCmd.PersistentFlags().StringP("operator-webhook-namespace", "t", "", "The namespace the services lives in (Optional, only needed inside kube)")
	rootCmd.PersistentFlags().BoolP("register", "r", true, "Register the extension")

	rootCmd.PersistentFlags().StringP("graceful-fail-time", "f", "", "Graceful fail time for eirini pods")
	rootCmd.PersistentFlags().StringP("graceful-success-time", "g", "", "Graceful success time for eirini pods")
	rootCmd.PersistentFlags().StringP("downloader-entrypoint", "d", "", "Eirini staging downloader entrypoint")
	rootCmd.PersistentFlags().StringP("executor-entrypoint", "e", "", "Eirini staging executor entrypoint")
	rootCmd.PersistentFlags().StringP("uploader-entrypoint", "u", "", "Eirini staging uploader entrypoint")
	rootCmd.PersistentFlags().StringP("opi-entrypoint", "o", "", "Eirini opi entrypoint")
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
	viper.SetDefault("OPERATOR_WEBHOOK_HOST", "")
	viper.SetDefault("OPERATOR_WEBHOOK_PORT", "")
	viper.SetDefault("OPERATOR_SERVICE_NAME", "")
	viper.SetDefault("OPERATOR_WEBHOOK_NAMESPACE", "")
	viper.SetDefault("EIRINI_EXTENSION_REGISTER", "")
	viper.SetDefault("GRACEFUL_FAIL_TIME", "")
	viper.SetDefault("GRACEFUL_SUCCESS_TIME", "")
	viper.SetDefault("DOWNLOADER_ENTRYPOINT", "")
	viper.SetDefault("EXECUTOR_ENTRYPOINT", "")
	viper.SetDefault("UPLOADER_ENTRYPOINT", "")
	viper.SetDefault("OPI_ENTRYPOINT", "")

	viper.BindEnv("namespace", "NAMESPACE")
	viper.BindEnv("loggregator-key-path", "LOGGREGATOR_KEY_PATH")
	viper.BindEnv("loggregator-endpoint", "LOGGREGATOR_ENDPOINT")
	viper.BindEnv("loggregator-ca-path", "LOGGREGATOR_CA_PATH")
	viper.BindEnv("loggregator-cert-path", "LOGGREGATOR_CERT_PATH")
	viper.BindEnv("operator-webhook-host", "OPERATOR_WEBHOOK_HOST")
	viper.BindEnv("operator-webhook-port", "OPERATOR_WEBHOOK_PORT")
	viper.BindEnv("operator-service-name", "OPERATOR_SERVICE_NAME")
	viper.BindEnv("operator-webhook-namespace", "OPERATOR_WEBHOOK_NAMESPACE")
	viper.BindEnv("register", "EIRINI_EXTENSION_REGISTER")
	viper.BindEnv("graceful-fail-time", "GRACEFUL_FAIL_TIME")
	viper.BindEnv("graceful-success-time", "GRACEFUL_SUCCESS_TIME")
	viper.BindEnv("downloader-entrypoint", "DOWNLOADER_ENTRYPOINT")
	viper.BindEnv("executor-entrypoint", "EXECUTOR_ENTRYPOINT")
	viper.BindEnv("uploader-entrypoint", "UPLOADER_ENTRYPOINT")
	viper.BindEnv("opi-entrypoint", "OPI_ENTRYPOINT")

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
