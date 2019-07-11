package podwatcher

import (
	config "github.com/SUSE/eirini-loggregator-bridge/config"
	"k8s.io/client-go/kubernetes"
)

type PodWatcher struct {
	Config     config.ConfigType
	kubeClient kubernetes.Interface
}

func NewPodWatcher(config config.ConfigType, kubeClient kubernetes.Interface) *PodWatcher {
	return &PodWatcher{Config: config, kubeClient: kubeClient}
}

func (pw *PodWatcher) Run() error {
	return nil
	// Consume a kubeclient
	// Create kube watcher

	// (save it somewhere?)
	// select on channels and handle events and spin up go routines for the new pod
	// Or stop goroutine for removed pods
	// Those goroutines read the logs  of the pod from the kube api and simply  writes metadata to a channel.
	// Then we have one or more reader instances that consumes the channel, converting metadata to loggregator envelopes and streams that to loggregator
}
