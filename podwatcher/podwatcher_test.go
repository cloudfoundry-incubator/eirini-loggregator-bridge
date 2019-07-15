package podwatcher_test

import (
	config "github.com/SUSE/eirini-loggregator-bridge/config"
	. "github.com/SUSE/eirini-loggregator-bridge/podwatcher"
	eirinixcatalog "github.com/SUSE/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("podwatcher", func() {
	catalog := eirinixcatalog.NewCatalog()
	Describe("PodWatcher Config", func() {
		Context("when initializing", func() {
			It("sets the config", func() {
				pw := NewPodWatcher(config.ConfigType{Namespace: "test"}, catalog.SimpleManager())
				cpw, ok := pw.(*PodWatcher)
				Expect(ok).To(BeTrue())
				Expect(cpw.Config).ToNot(BeNil())
				Expect(cpw.Config.Namespace).To(Equal("test"))
			})
		})
	})

	Describe("ContainerList", func() {

		var cl *ContainerList
		var pod *corev1.Pod
		BeforeEach(func() {
			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{UID: types.UID("poduid")},
				Spec:       corev1.PodSpec{Containers: []corev1.Container{{}}},
				Status:     corev1.PodStatus{},
			}
			cl = &ContainerList{Containers: map[string]*Container{}}
		})

		Context("when containers are running", func() {
			BeforeEach(func() {
				pod.Spec.Containers = []corev1.Container{
					{Name: "testcontainer"},
				}
				pod.Spec.InitContainers = []corev1.Container{
					{Name: "testinitcontainer"},
				}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "testcontainer",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
				pod.Status.InitContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "testinitcontainer",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
			})

			It("Adds the container in the containerlist", func() {
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				cont, ok := cl.GetContainer("poduid-testcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.Name).To(Equal("testcontainer"))
				cont, ok = cl.GetContainer("poduid-testinitcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.Name).To(Equal("testinitcontainer"))
			})
		})

		Context("when more containers for the same pod are added", func() {
			BeforeEach(func() {
				pod.Spec.Containers = []corev1.Container{
					{Name: "testcontainer"},
				}
				pod.Spec.InitContainers = []corev1.Container{
					{Name: "testinitcontainer"},
				}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "testcontainer",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
				pod.Status.InitContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "testinitcontainer",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
			})

			It("Adds the container in the containerlist", func() {
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				cont, ok := cl.GetContainer("poduid-testcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.Name).To(Equal("testcontainer"))
				cont, ok = cl.GetContainer("poduid-testinitcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.Name).To(Equal("testinitcontainer"))
			})
		})

		Context("when containers are added but are not running", func() {
			BeforeEach(func() {
				cl.Containers = map[string]*Container{
					"poduid-mycontainer": {
						Name: "MyContainer",
						UID:  "myContainerUID",
					},
					"poduid-myinitcontainer": {
						Name:          "MyInitContainer",
						UID:           "myInitContainerUID",
						InitContainer: true,
					},
				}

				pod.Spec.Containers = []corev1.Container{
					{Name: "mycontainer"},
					{Name: "mycontainer2"},
				}
				pod.Spec.InitContainers = []corev1.Container{
					{Name: "myinitcontainer"},
					{Name: "myinitcontainer2"},
				}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "mycontainer",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
					{
						Name: "mycontainer2",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
				pod.Status.InitContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "myinitcontainer",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
					{
						Name: "myinitcontainer2",
						State: corev1.ContainerState{
							Running: &corev1.ContainerStateRunning{},
						},
					},
				}
			})

			It("does not add the containers in the containerlist", func() {
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				_, ok := cl.GetContainer("poduid-mycontainer")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("poduid-mycontainer2")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("poduid-myinitcontainer")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("poduid-myinitcontainer2")
				Expect(ok).Should(BeTrue())
			})
		})

		Context("when containers are completely removed", func() {
			BeforeEach(func() {
				cl.Containers = map[string]*Container{
					"myContainerUID": {
						Name: "MyContainer",
						UID:  "myContainerUID",
					},
					"myInitContainerUID": {
						Name:          "MyInitContainer",
						UID:           "myInitContainerUID",
						InitContainer: true,
					},
				}

				// The container doesn't exist in the pod we get with the Event
				pod.Spec.Containers = []corev1.Container{}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{}
			})

			It("Removes the container from the containerlist", func() {
				_, ok := cl.GetContainer("myContainerUID")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("myInitContainerUID")
				Expect(ok).Should(BeTrue())
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				_, ok = cl.GetContainer("myContainerUID")
				Expect(ok).Should(BeFalse())
				_, ok = cl.GetContainer("myInitContainerUID")
				Expect(ok).Should(BeFalse())
			})
		})

		Context("when containers don't have status", func() {
			BeforeEach(func() {
				cl.Containers = map[string]*Container{
					"poduid-mycontainer": {
						Name: "MyContainer",
						UID:  "myContainerUID",
					},
					"poduid-myinitcontainer": {
						Name:          "MyInitContainer",
						UID:           "myInitContainerUID",
						InitContainer: true,
					},
				}

				// The container exist in the pod we get with the Event but doesn't has
				// a status.
				pod.Spec.Containers = []corev1.Container{
					{Name: "mycontainer"},
				}
				pod.Spec.InitContainers = []corev1.Container{
					{Name: "myinitcontainer"},
				}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{}
			})

			It("Removes the container from the containerlist", func() {
				_, ok := cl.GetContainer("poduid-mycontainer")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("poduid-myinitcontainer")
				Expect(ok).Should(BeTrue())
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				_, ok = cl.GetContainer("poduid-mycontainer")
				Expect(ok).Should(BeFalse())
				_, ok = cl.GetContainer("poduid-myinitcontainer")
				Expect(ok).Should(BeFalse())
			})
		})

		Context("when containers have a non-running status", func() {
			BeforeEach(func() {
				cl.Containers = map[string]*Container{
					"poduid-mycontainer": {
						Name: "MyContainer",
						UID:  "myContainerUID",
					},
					"poduid-myinitcontainer": {
						Name:          "MyInitContainer",
						UID:           "myInitContainerUID",
						InitContainer: true,
					},
				}

				// The container exist in the pod we get with the Event but doesn't has
				// a status.
				pod.Spec.Containers = []corev1.Container{
					{Name: "mycontainer"},
				}
				pod.Spec.InitContainers = []corev1.Container{
					{Name: "myinitcontainer"},
				}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "myinitcontainer",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{},
						},
					},
					{
						Name: "mycontainer",
						State: corev1.ContainerState{
							Waiting: &corev1.ContainerStateWaiting{},
						},
					},
				}
			})

			It("Removes the container from the containerlist", func() {
				_, ok := cl.GetContainer("poduid-mycontainer")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("poduid-myinitcontainer")
				Expect(ok).Should(BeTrue())
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				_, ok = cl.GetContainer("poduid-mycontainer")
				Expect(ok).Should(BeFalse())
				_, ok = cl.GetContainer("poduid-myinitcontainer")
				Expect(ok).Should(BeFalse())
			})
		})
	})
})
