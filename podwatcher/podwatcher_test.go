package podwatcher_test

import (
	config "github.com/SUSE/eirini-loggregator-bridge/config"
	. "github.com/SUSE/eirini-loggregator-bridge/podwatcher"
	fakeKubeClientset "github.com/SUSE/eirini-loggregator-bridge/podwatcher/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

var _ = Describe("podwatcher", func() {

	Describe("PodWatcher Config", func() {
		Context("when initializing", func() {
			It("sets the config", func() {
				pw := NewPodWatcher(config.ConfigType{Namespace: "test"}, &fakeKubeClientset.FakeInterface{})
				Expect(pw.Config).ToNot(BeNil())
				Expect(pw.Config.Namespace).To(Equal("test"))
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
			cl = &ContainerList{}
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
		})

		Context("when containers are added (sidecars)", func() {
		})

		Context("when container is removed", func() {})

		Context("when initcontainers are running, and containers are not", func() {
		})

	})

})
