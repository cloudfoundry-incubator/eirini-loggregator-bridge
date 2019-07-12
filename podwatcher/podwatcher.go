package podwatcher

import (
	"fmt"
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

func (cl *ContainerList) GetContainer(name string) (Container, bool) {
	c, ok := (*cl)[name]
	return c, ok
}

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

func generateContainerUID(pod *corev1.Pod, container corev1.Container) string {
	return fmt.Sprintf("%s-%s", string(pod.UID), container.Name)
}

func findContainerState(name string, containerStatuses []corev1.ContainerStatus) *corev1.ContainerState {
	for _, status := range containerStatuses {
		if status.Name == name {
			return &status.State
		}
	}
	return nil
}

// EnsurePodStatus handles a pod event by adding or removing container tailing
// goroutines. Every running container in the monitored namespace has its own
// goroutine that reads its log stream. When a container is stopped we stop
// the relevant gorouting (if it is still running, it could already be stopped
// because of an error).
func (cl ContainerList) EnsurePodStatus(pod *corev1.Pod) error {
	//LogDebug(pod.Status.ContainerStatuses)
	//LogDebug(pod.Status.InitContainerStatuses)
	for _, c := range pod.Spec.InitContainers {
		cUID := generateContainerUID(pod, c)
		cState := findContainerState(c.Name, pod.Status.InitContainerStatuses)
		//		LogDebug("status:", pod.Status.InitContainerStatuses[k].State)
		if cState != nil && cState.Running != nil {
			LogDebug(cUID + " init container is running - ensure  we are streaming")
		} else {
			LogDebug(cUID + " init container is not running, ensure we are NOT streaming")
		}

	}
	for _, c := range pod.Spec.Containers {
		cUID := generateContainerUID(pod, c)
		cState := findContainerState(c.Name, pod.Status.ContainerStatuses)
		if cState != nil && cState.Running != nil {
			LogDebug(cUID + " container is running - ensure  we are streaming")
		} else {
			LogDebug(cUID + " container is not running, ensure we are NOT streaming")
		}

	}
	return nil
}

type Container struct {
	killChannel chan bool
	PodName     string
	Namespace   string
	Name        string
	PodUID      string
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

	containers := ContainerList{}
	for { // Keep reading the result channel for new events
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

			containers.EnsurePodStatus(pod)
		}
	}

	return nil

	// TODO:
	// - Consume a kubeclient ✓
	// - Create kube watcher ✓
	// - Select on channels and handle events and spin up go routines for the new pod
	//   Or stop goroutine for removed pods
	//   Those goroutines read the logs  of the pod from the kube api and simply  writes metadata to a channel.
	// - Then we have one or more reader instances that consumes the channel, converting metadata to loggregator envelopes and streams that to loggregator
}
