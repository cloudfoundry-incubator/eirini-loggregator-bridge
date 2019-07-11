package podwatcher

import (
	config "github.com/SUSE/eirini-loggregator-bridge/config"
	. "github.com/SUSE/eirini-loggregator-bridge/logger"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
)

type PodWatcher struct {
	Config     config.ConfigType
	kubeClient kubernetes.Interface
	containers []Container
}

type ContainerList map[string]Container

//TODO:
// AddContainer adds a container to the list if doesn't exist,
// or kills the previously one by using the killChannel and adds the new one to the list
func (cl ContainerList) AddContainer(c Container) error {
	// TODO: Fix this
	LogDebug("Adding container ", c)

	return nil
}

func (cl ContainerList) RemovePODContainers(podUID string) error {
	// TODO: Fix this, and remove all containers belonging to a POD
	LogDebug("Removing POD's containers ", podUID)

	return nil
}

type Container struct {
	killChannel   chan bool
	PodName       string
	Namespace     string
	ContainerName string
	PodUID        string
}

func NewPodWatcher(config config.ConfigType, kubeClient kubernetes.Interface) *PodWatcher {
	return &PodWatcher{
		Config:     config,
		kubeClient: kubeClient,
	}
}

func (pw *PodWatcher) GenWatcher(namespace string) (watch.Interface, error) {
	podInterface := pw.kubeClient.CoreV1().Pods(namespace)

	watcher, err := podInterface.Watch(
		metav1.ListOptions{Watch: true})
	return watcher, err
}

func (pw *PodWatcher) Run() error {
	watcher, err := pw.GenWatcher(pw.Config.Namespace)
	if err != nil {
		return errors.Wrap(err, "failed to set up watch")
	}

	//added := make(chan *Container)
	//removed := make(chan *Container)

	// Keep reading the result channel for new events
	for {
		select {
		case e := <-watcher.ResultChan():
			if e.Object == nil {
				// Closed because of error
				// TODO: Handle errors ( maybe kill the whole application )
				// because it is going to run in goroutines, and we can't
				// just return gracefully and panicking the whole
				return errors.New("no object returned from watcher")
			}

			pod, ok := e.Object.(*corev1.Pod)
			if !ok {
				LogDebug(errors.New("Received non-pod object in watcher channel"))
				continue
			}

			containers := ContainerList{}

			switch e.Type {
			case watch.Added, watch.Modified:
				LogDebug("POD received from the watcher", pod)

				// We need also to loop over InitContainers (staging?)
				for _, c := range pod.Spec.Containers {
					containers.AddContainer(Container{
						Namespace:     pod.Namespace,
						PodName:       pod.Name,
						ContainerName: c.Name,
						PodUID:        string(pod.UID),
						killChannel:   make(chan bool),
					})
				}
			case watch.Deleted:
				containers.RemovePODContainers(string(pod.UID))
			default:
				LogDebug("Unprocessable watch event", e.Type)
			}
		}
	}

	return nil

	// Consume a kubeclient ✓
	// Create kube watcher

	// (save it somewhere?)
	// select on channels and handle events and spin up go routines for the new pod
	// Or stop goroutine for removed pods
	// Those goroutines read the logs  of the pod from the kube api and simply  writes metadata to a channel.
	// Then we have one or more reader instances that consumes the channel, converting metadata to loggregator envelopes and streams that to loggregator
}
