package podwatcher

import (
	"bufio"
	"context"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	flowcontrol "k8s.io/client-go/util/flowcontrol"

	"code.cloudfoundry.org/eirini-loggregator-bridge/config"
	. "code.cloudfoundry.org/eirini-loggregator-bridge/logger"
	"code.cloudfoundry.org/go-loggregator/v8"
	"code.cloudfoundry.org/go-loggregator/v8/rpc/loggregator_v2"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type LoggregatorAppMeta struct {
	SourceID, InstanceID                               string
	SourceType, PodName, Namespace, Container, Cluster string // Custom tags
}

type Loggregator struct {
	Context           context.Context
	Meta              *LoggregatorAppMeta
	ConnectionOptions config.LoggregatorOptions
	KubeClient        *kubernetes.Clientset
	LoggregatorClient *loggregator.IngressClient
	KubeConfig        *rest.Config
}

type LoggregatorLogger struct{}

func (LoggregatorLogger) Printf(message string, args ...interface{}) {
	LogDebug(append([]interface{}{message}, args...))
}
func (LoggregatorLogger) Panicf(message string, args ...interface{}) {
	panic(message)
}

func NewLoggregator(ctx context.Context, m *LoggregatorAppMeta, kubeClient *kubernetes.Clientset, kubeConfig *rest.Config, connectionOptions config.LoggregatorOptions) *Loggregator {
	return &Loggregator{Meta: m, KubeClient: kubeClient, ConnectionOptions: connectionOptions, Context: ctx, KubeConfig: kubeConfig}
}

func (l *Loggregator) Envelope(message []byte) *loggregator_v2.Envelope {
	LogDebug("Creating envelope for string: ", string(message))

	return &loggregator_v2.Envelope{
		Message: &loggregator_v2.Envelope_Log{
			Log: &loggregator_v2.Log{
				Payload: message,
				Type:    loggregator_v2.Log_OUT,
			},
		},
		SourceId:   l.Meta.SourceID,
		InstanceId: l.Meta.InstanceID,
		Tags: map[string]string{
			"source_type": l.Meta.SourceType,
			"pod_name":    l.Meta.PodName,
			"namespace":   l.Meta.Namespace,
			"container":   l.Meta.Container,
			"cluster":     l.Meta.Cluster, // ??
		},
		Timestamp: time.Now().Unix() * 1000000000,
	}
}

func (l *Loggregator) SetupLoggregatorClient() error {
	tlsConfig, err := loggregator.NewIngressTLSConfig(
		l.ConnectionOptions.CAPath,
		l.ConnectionOptions.CertPath,
		l.ConnectionOptions.KeyPath,
	)
	if err != nil {
		return err
	}

	logger := LoggregatorLogger{}

	loggregatorClient, err := loggregator.NewIngressClient(
		tlsConfig,
		// Temporary make flushing more frequent to be able to debug
		loggregator.WithBatchMaxSize(uint(100)),
		loggregator.WithLogger(logger),
		loggregator.WithAddr(l.ConnectionOptions.Endpoint),
	)

	if err != nil {
		return err
	}

	l.LoggregatorClient = loggregatorClient
	return nil
}

func (l *Loggregator) Write(b []byte) (int, error) {
	l.LoggregatorClient.Emit(l.Envelope(b))

	return len(b), nil
}

func (l *Loggregator) Tail(namespace, pod, container string) error {
	configShallowCopy := *l.KubeConfig
	configShallowCopy.RateLimiter = flowcontrol.NewTokenBucketRateLimiter(configShallowCopy.QPS, configShallowCopy.Burst)

	kubeClient, err := kubernetes.NewForConfig(&configShallowCopy)
	if err != nil {
		return errors.Wrap(err, "failed creating kubeClient")
	}

	podData, err := kubeClient.CoreV1().Pods(namespace).Get(l.Context, pod, metav1.GetOptions{})
	if err != nil {
		return err
	}

	follow := true
	previous := false

	// XXX: TODO inspect pod phase instead of container statuses.
	//	podData.
	containers := ExtractContainersFromPod(podData)
	for _, c := range containers {
		if c.Name == container {
			if c.State != nil && c.State.Terminated != nil {
				LogDebug("Grabbing logs only from terminated pod")
				follow = false
				previous = true
			}
		}
	}

	req := kubeClient.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Name(pod).
		Resource("pods").
		SubResource("log").
		Param("follow", strconv.FormatBool(follow)).
		Param("container", container).
		Param("previous", strconv.FormatBool(previous)).
		Param("timestamps", strconv.FormatBool(false))
	stream, err := req.Stream(l.Context)
	if err != nil {
		return err
	}

	defer stream.Close()
	reader := bufio.NewReader(stream)
	for {
		line, err := reader.ReadBytes('\n')
		if err == io.EOF {
			break
		}

		if err != nil {
			return err
		}

		_, err = l.Write([]byte(strings.TrimSpace(string(line))))
		if err != nil {
			return err
		}
	}

	return nil
}
