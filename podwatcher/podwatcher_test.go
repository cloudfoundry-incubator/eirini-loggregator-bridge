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
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{
					{
						Name: "testcontainer",
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
			})
		})

		Context("when containers aren't running", func() {
			BeforeEach(func() {
				cl.Containers = map[string]*Container{
					"myContainerUID": {
						Name: "MyContainer",
						UID:  "myContainerUID",
					},
				}

				// The container doesn't exist in the pod we get with the Event
				pod.Spec.Containers = []corev1.Container{}
				pod.Status.ContainerStatuses = []corev1.ContainerStatus{}
			})

			It("Removes the container from the containerlist", func() {
				_, ok := cl.GetContainer("myContainerUID")
				Expect(ok).Should(BeTrue())
				err := cl.EnsurePodStatus(pod)
				Expect(err).To(BeNil())
				_, ok = cl.GetContainer("myContainerUID")
				Expect(ok).Should(BeFalse())
			})
		})

		Context("when containers are added (sidecars)", func() {
		})

		Context("when container is removed", func() {})

		Context("when initcontainers are running, and containers are not", func() {
		})

	})

})
