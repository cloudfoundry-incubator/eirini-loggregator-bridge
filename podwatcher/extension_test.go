package podwatcher_test

import (
	"context"
	"encoding/json"

	. "code.cloudfoundry.org/eirini-loggregator-bridge/podwatcher"
	eirinixcatalog "code.cloudfoundry.org/eirinix/testing"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	admissionv1beta1 "k8s.io/api/admission/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func decodePatches(resp admission.Response) string {
	var r string
	for _, patch := range resp.Patches {
		r += patch.Json()
	}
	return r
}

const (
	addOpiPatch = `{"op":"add","path":"/spec/containers/0/command","value":["dumb-init","--","/bin/sh","-c","(  /lifecycle/launch \u0026\u0026 sleep 5 ) || sleep 5"]}`
)

var _ = Describe("Eirini extension", func() {
	eirinixcat := eirinixcatalog.NewCatalog()
	gracefulInjector := NewgracePeriodInjector(&GraceOptions{})
	eiriniManager := eirinixcat.SimpleManager()
	request := admission.Request{}
	pod := &corev1.Pod{
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name: "opi",
				},
			},
		},
	}

	BeforeEach(func() {
		gracefulInjector = NewgracePeriodInjector(&GraceOptions{})
		eirinixcat = eirinixcatalog.NewCatalog()
		gracefulInjector = NewgracePeriodInjector(&GraceOptions{})
		eiriniManager = eirinixcat.SimpleManager()

		raw, err := json.Marshal(pod)
		if err != nil {
			Expect(err).To(BeNil())
		}

		request = admission.Request{AdmissionRequest: admissionv1beta1.AdmissionRequest{Object: runtime.RawExtension{Raw: raw}}}
	})

	Describe("GracePeriod Injector", func() {
		Context("when initializing", func() {
			It("sets the default config", func() {
				Expect(gracefulInjector.Options.FailGracePeriod).To(Equal("5"))
				Expect(gracefulInjector.Options.SuccessGracePeriod).To(Equal("5"))
				Expect(gracefulInjector.Options.StagingDownloaderEntrypoint).To(Equal("/packs/downloader"))
				Expect(gracefulInjector.Options.StagingUploaderEntrypoint).To(Equal("/packs/uploader"))
				Expect(gracefulInjector.Options.StagingExecutorEntrypoint).To(Equal("/packs/executor"))
				Expect(gracefulInjector.Options.RuntimeEntrypoint).To(Equal("/lifecycle/launch"))
			})
		})
	})

	Describe("GracePeriod Injector", func() {
		Context("when handling a runtime app", func() {
			It("Injects a grace period", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(addOpiPatch))
			})
		})
	})

})
