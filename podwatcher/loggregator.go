package podwatcher

import (
	"code.cloudfoundry.org/go-loggregator"
	"code.cloudfoundry.org/go-loggregator/rpc/loggregator_v2"
	"github.com/SUSE/eirini-loggregator-bridge/config"
	"io"
	"k8s.io/client-go/kubernetes"
	"strconv"
	"time"
)

type LoggregatorAppMeta struct {
	SourceID, InstanceID                               string
	SourceType, PodName, Namespace, Container, Cluster string // Custom tags
}

type Loggregator struct {
	Meta              *LoggregatorAppMeta
	ConnectionOptions config.LoggregatorOptions
	KubeClient        *kubernetes.Clientset
	LoggregatorClient *loggregator.IngressClient
}

func NewLoggregator(m *LoggregatorAppMeta, kubeClient *kubernetes.Clientset, connectionOptions config.LoggregatorOptions) *Loggregator {
	return &Loggregator{Meta: m, KubeClient: kubeClient, ConnectionOptions: connectionOptions}
}

func (l *Loggregator) Envelope(message []byte) *loggregator_v2.Envelope {
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

	loggregatorClient, err := loggregator.NewIngressClient(
		tlsConfig,
		// Temporary make flushing more frequent to be able to debug
		loggregator.WithBatchMaxSize(uint(100)),
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
	req := l.KubeClient.CoreV1().RESTClient().Get().
		Namespace(namespace).
		Name(pod).
		Resource("pods").
		SubResource("log").
		Param("follow", strconv.FormatBool(true)).
		Param("container", container).
		Param("previous", strconv.FormatBool(false)).
		Param("timestamps", strconv.FormatBool(false))
	readCloser, err := req.Stream()
	if err != nil {
		return err
	}

	defer readCloser.Close()
	_, err = io.Copy(l, readCloser)
	if err != nil {
		return err
	}

	return nil
}
