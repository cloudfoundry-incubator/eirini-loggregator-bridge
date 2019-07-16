package podwatcher

import (
	"fmt"
	"sync"

	config "github.com/SUSE/eirini-loggregator-bridge/config"
	. "github.com/SUSE/eirini-loggregator-bridge/logger"
	eirinix "github.com/SUSE/eirinix"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type PodWatcher struct {
	Config     config.ConfigType
	Containers ContainerList
	Manager    eirinix.Manager
}

type Container struct {
	killChannel        chan bool
	PodName            string
	Namespace          string
	Name               string
	PodUID             string
	UID                string
	InitContainer      bool
	State              *corev1.ContainerState
	LoggregatorOptions config.LoggregatorOptions
	Loggregator        *Loggregator
	AppMeta            *LoggregatorAppMeta
}

type ContainerList struct {
	Containers         map[string]*Container
	KubeConfig         *rest.Config
	LoggregatorOptions config.LoggregatorOptions
	Tails              sync.WaitGroup
}

func (cl *ContainerList) GetContainer(uid string) (*Container, bool) {
	c, ok := cl.Containers[uid]
	return c, ok
}

func (cl *ContainerList) AddContainer(c *Container) {
	cl.Containers[c.UID] = c
	c.Read(cl.LoggregatorOptions, cl.KubeConfig, &cl.Tails)
}

func (cl *ContainerList) RemoveContainer(uid string) error {
	_, ok := cl.GetContainer(uid)
	if ok {
		// TODO: Cleanup goroutine with killChannel
	}
	delete(cl.Containers, uid)
	return nil
}

// EnsureContainer make sure the container exists in the list and we are
// monitoring it.
func (cl ContainerList) EnsureContainer(c *Container) error {
	// TODO: implement this
	LogDebug(c.UID + ": ensuring container is monitored")

	if _, ok := cl.GetContainer(c.UID); !ok {
		cl.AddContainer(c)
	}
	return nil
}

func (c *Container) Read(LoggregatorOptions config.LoggregatorOptions, KubeConfig *rest.Config, wg *sync.WaitGroup) {
	wg.Add(1)

	go func(c *Container, w *sync.WaitGroup) {
		defer wg.Done()

		kubeClient, err := kubernetes.NewForConfig(KubeConfig)
		if err != nil {
			LogError(err.Error())
		}
		c.Loggregator = NewLoggregator(c.AppMeta, kubeClient, LoggregatorOptions)
		err = c.Tail(kubeClient)
		if err != nil {
			LogError("Error: ", err.Error())
		}
	}(c, wg)
}

// Tail connects to the Kube
func (c *Container) Tail(kubeClient *kubernetes.Clientset) error {
	// NOTE: We may end up implementing a cursor to get
	// log parts as we might have log losses due to the watcher
	// starting up late.
	return c.Loggregator.Tail(c.Namespace, c.PodName, c.Name)
}

func (c *Container) generateUID() {
	c.UID = fmt.Sprintf("%s-%s", string(c.PodUID), c.Name)
}

func (c *Container) findState(containerStatuses []corev1.ContainerStatus) {
	for _, status := range containerStatuses {
		if status.Name == c.Name {
			c.State = &status.State
		}
	}
}

// cleanup removes containers from the containerlist if they don't exist in the given
// map. This should be used to remove leftover containers from our containerlist
// when they disappear from the pod. existingContainers should be all containers
// of the same pod!
func (cl *ContainerList) cleanup(podUID string, existingPodContainers map[string]*Container) {
	// Remove only containers for the given pod
	for _, c := range cl.Containers {
		if _, ok := existingPodContainers[c.UID]; c.PodUID == podUID && !ok {
			cl.RemoveContainer(c.UID)
		}
	}
}

// UpdateContainer decides whether a container should be added, left alone
// or removed from the container list. It does that but checking the state of
// of the container.
func (cl *ContainerList) UpdateContainer(c *Container) error {
	if c.State != nil && c.State.Running != nil {
		cl.EnsureContainer(c)
	} else {
		err := cl.RemoveContainer(c.UID)
		if err != nil {
			return err
		}
	}
	return nil
}

func ExtractContainersFromPod(pod *corev1.Pod) map[string]*Container {

	sourceType, ok := pod.GetLabels()["source_type"]
	if ok && sourceType == "APP" {
		sourceType = "APP/PROC/WEB"
	}

	result := map[string]*Container{}
	// NOTE: The order of the lists matter!
	for i, clist := range [][]corev1.Container{pod.Spec.InitContainers, pod.Spec.Containers} {
		cstatuses := [][]corev1.ContainerStatus{pod.Status.InitContainerStatuses, pod.Status.ContainerStatuses}
		for _, c := range clist {
			container := &Container{
				Name:          c.Name,
				PodName:       pod.Name,
				PodUID:        string(pod.UID),
				Namespace:     pod.Namespace,
				killChannel:   make(chan bool),
				InitContainer: (i == 0),

				AppMeta: &LoggregatorAppMeta{
					SourceID:   pod.GetLabels()["guid"], // TODO: Handle the case when this is empty (when ? when its not an eirini app. Maybe filter for guids in eirinix?)
					InstanceID: "0",                     // parse it from pod name( app-1-blabla)
					SourceType: sourceType,
					PodName:    pod.Name,
					Namespace:  pod.Namespace,
					Container:  c.Name,
					// TODO: Is this correct?
					// https://github.com/gdankov/loggregator-ci/blob/eirini/docker-images/fluentd/plugins/loggregator.rb#L54
					Cluster: pod.GetClusterName(),
				},
			}
			container.generateUID()
			container.findState(cstatuses[i])
			result[container.UID] = container
		}
	}
	return result
}

// EnsurePodStatus handles a pod event by adding or removing container tailing
// goroutines. Every running container in the monitored namespace has its own
// goroutine that reads its log stream. When a container is stopped we stop
// the relevant gorouting (if it is still running, it could already be stopped
// because of an error).
func (cl *ContainerList) EnsurePodStatus(pod *corev1.Pod) error {
	podContainers := ExtractContainersFromPod(pod)

	for _, c := range podContainers {
		cl.UpdateContainer(c)
	}

	cl.cleanup(string(pod.UID), podContainers)

	return nil
}

func NewPodWatcher(config config.ConfigType) eirinix.Watcher {
	return &PodWatcher{
		Config:     config,
		Containers: ContainerList{Containers: map[string]*Container{}},
	}
}

func (pw *PodWatcher) Finish() {
	pw.Containers.Tails.Wait()
}

func (pw *PodWatcher) Handle(manager eirinix.Manager, e watch.Event) {
	manager.GetLogger().Debug("Received event: ", e)
	if e.Object == nil {
		// Closed because of error
		// TODO: Handle errors ( maybe kill the whole application )
		// because it is going to run in goroutines, and we can't
		// just return gracefully and panicking the whole
		return
	}

	pod, ok := e.Object.(*corev1.Pod)
	if !ok {
		LogError("Received non-pod object in watcher channel")
		return
	}
	config, err := manager.GetKubeConnection()
	if err != nil {
		LogError(err.Error())
		return
	}
	pw.Containers.KubeConfig = config
	pw.Containers.LoggregatorOptions = pw.Config.GetLoggregatorOptions()
	pw.Containers.EnsurePodStatus(pod)

	// TODO:
	// - Consume a kubeclient ✓ -> moved to eirinix
	// - Create kube watcher ✓ -> moved to eirinix
	// - Select on channels and handle events and spin up go routines for the new pod
	//   Or stop goroutine for removed pods
	//   Those goroutines read the logs  of the pod from the kube api and simply  writes metadata to a channel.
	// - Then we have one or more reader instances that consumes the channel, converting metadata to loggregator envelopes and streams that to loggregator
}
