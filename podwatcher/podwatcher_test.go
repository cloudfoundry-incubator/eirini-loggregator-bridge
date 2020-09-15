package podwatcher_test

import (
	config "code.cloudfoundry.org/eirini-loggregator-bridge/config"
	. "code.cloudfoundry.org/eirini-loggregator-bridge/podwatcher"
	eirinix "code.cloudfoundry.org/eirinix"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/rest"
)

var _ = Describe("podwatcher", func() {
	cl := &ContainerList{}

	BeforeEach(func() {
		cl = &ContainerList{KubeConfig: &rest.Config{}}
	})
	AfterEach(func() { cl.Tails.Wait() })

	Describe("PodWatcher Config", func() {
		Context("when initializing", func() {
			It("sets the config", func() {
				pw := NewPodWatcher(config.ConfigType{Namespace: "test"})
				Expect(pw.Config).ToNot(BeNil())
				Expect(pw.Config.Namespace).To(Equal("test"))
			})
		})
	})

	Describe("ContainerList", func() {
		var pod *corev1.Pod
		BeforeEach(func() {
			pod = &corev1.Pod{
				ObjectMeta: metav1.ObjectMeta{
					UID:    types.UID("poduid"),
					Name:   "ruby-app-tmp-c6858e2e56-2",
					Labels: map[string]string{eirinix.LabelAppGUID: "app-guid", eirinix.LabelSourceType: "APP"},
				},
				Spec:   corev1.PodSpec{Containers: []corev1.Container{{}}},
				Status: corev1.PodStatus{},
			}
			cl.Containers = map[string]*Container{}
		})
		AfterEach(func() { cl.Tails.Wait() })

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
				Expect(cont.AppMeta.SourceType).To(Equal("APP/PROC/WEB"))
			})

			It("Sets the SourceType correctly when source_type is APP", func() {
				pod.ObjectMeta.Labels[eirinix.LabelSourceType] = "APP"
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				cont, ok := cl.GetContainer("poduid-testinitcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.AppMeta.SourceType).To(Equal("APP/PROC/WEB"))
			})

			It("Sets the SourceType correctly when source_type it is not APP", func() {
				pod.ObjectMeta.Labels[eirinix.LabelSourceType] = "somethingelse"
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				cont, ok := cl.GetContainer("poduid-testinitcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.AppMeta.SourceType).To(Equal("somethingelse"))
			})

			It("Sets the InstanceID to the one extraced from the pod name", func() {
				pod.ObjectMeta.Name = "ruby-app-tmp-c6858e2e56-2"
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				cont, ok := cl.GetContainer("poduid-testinitcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.Name).To(Equal("testinitcontainer"))
				Expect(cont.AppMeta.InstanceID).To(Equal("2"))
			})

			It("Sets the InstanceID to 0 for pods with no instance id in the name (e.g. staging)", func() {
				pod.ObjectMeta.Name = "6ad9f634-b32e-4890-b1ba-55202d95bc3a-xdcp6"
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				cont, ok := cl.GetContainer("poduid-testinitcontainer")
				Expect(ok).Should(BeTrue())
				Expect(cont.Name).To(Equal("testinitcontainer"))
				Expect(cont.AppMeta.InstanceID).To(Equal("0"))
			})

			It("Doesn't add any containers if the guid is empty", func() {
				delete(pod.ObjectMeta.Labels, eirinix.LabelAppGUID)
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				Expect(len(cl.Containers)).Should(Equal(0))
			})

		})

		Context("when more containers for the same pod are added", func() {
			AfterEach(func() { cl.Tails.Wait() })
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
			AfterEach(func() { cl.Tails.Wait() })
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
			AfterEach(func() { cl.Tails.Wait() })
			BeforeEach(func() {
				cl.Containers = map[string]*Container{
					"podContainerUID": {
						Name:   "MyContainer",
						UID:    "podContainerUID",
						PodUID: string(pod.UID),
					},
					"otherPodContainerUID": {
						Name:   "MyContainer",
						UID:    "otherPodContainerUID",
						PodUID: "someOtherPodUID",
					},
					"podInitContainerUID": {
						Name:          "MyInitContainer",
						UID:           "podInitContainerUID",
						InitContainer: true,
						PodUID:        string(pod.UID),
					},
					"otherPodInitContainerUID": {
						Name:          "MyInitContainer",
						UID:           "otherPodInitContainerUID",
						InitContainer: true,
						PodUID:        "someOtherPodUID",
					},
				}

				// The container doesn't exist in the pod we get with the Event
				pod.Spec.Containers = []corev1.Container{}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{}
			})

			It("Removes the pod's containers from the containerlist", func() {
				_, ok := cl.GetContainer("podContainerUID")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("otherPodContainerUID")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("podInitContainerUID")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("otherPodInitContainerUID")
				Expect(ok).Should(BeTrue())

				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())

				_, ok = cl.GetContainer("podContainerUID")
				Expect(ok).Should(BeFalse())
				_, ok = cl.GetContainer("podInitContainerUID")
				Expect(ok).Should(BeFalse())
				_, ok = cl.GetContainer("otherPodContainerUID")
				Expect(ok).Should(BeTrue())
				_, ok = cl.GetContainer("otherPodInitContainerUID")
				Expect(ok).Should(BeTrue())
			})
		})

		Context("when containers don't have status", func() {
			AfterEach(func() { cl.Tails.Wait() })
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
			AfterEach(func() { cl.Tails.Wait() })
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
