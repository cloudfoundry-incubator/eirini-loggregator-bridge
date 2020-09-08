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

func jsonifyPatches(resp admission.Response) []string {
	var r []string
	for _, patch := range resp.Patches {
		r = append(r, patch.Json())
	}
	return r
}

const (
	addOpiPatch                = `{"op":"add","path":"/spec/containers/0/command","value":["dumb-init","--","/bin/sh","-c","(  /lifecycle/launch \u0026\u0026 sleep 5 ) || sleep 5"]}`
	addUploaderPatch           = `{"op":"add","path":"/spec/containers/0/command","value":["/bin/sh","-c","( /packs/uploader \u0026\u0026 sleep 5 ) || sleep 5"]}`
	addDownloaderPatch         = `{"op":"add","path":"/spec/initContainers/0/command","value":["/bin/sh","-c","( /packs/downloader \u0026\u0026 sleep 5 ) || sleep 5"]}`
	addExecutorPatch           = `{"op":"add","path":"/spec/initContainers/0/command","value":["/bin/sh","-c","( /packs/executor \u0026\u0026 sleep 5 ) || sleep 5"]}`
	stagingFullPatchUploader   = `{"op":"add","path":"/spec/containers/0/command","value":["/bin/sh","-c","( /packs/uploader \u0026\u0026 sleep 5 ) || sleep 5"]}`
	stagingFullPatchExecutor   = `{"op":"add","path":"/spec/initContainers/0/command","value":["/bin/sh","-c","( /packs/executor \u0026\u0026 sleep 5 ) || sleep 5"]}`
	stagingFullPatchDownloader = `{"op":"add","path":"/spec/initContainers/1/command","value":["/bin/sh","-c","( /packs/downloader \u0026\u0026 sleep 5 ) || sleep 5"]}`
)

var _ = Describe("Eirini extension", func() {
	eirinixcat := eirinixcatalog.NewCatalog()
	gracefulInjector := NewGracePeriodInjector(&GraceOptions{})
	eiriniManager := eirinixcat.SimpleManager()
	request := admission.Request{}
	pod := &corev1.Pod{}

	JustBeforeEach(func() {
		gracefulInjector = NewGracePeriodInjector(&GraceOptions{})
		eirinixcat = eirinixcatalog.NewCatalog()
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

			It("sets config", func() {
				gracefulInjector = NewGracePeriodInjector(&GraceOptions{
					FailGracePeriod:             "12",
					SuccessGracePeriod:          "90",
					StagingDownloaderEntrypoint: "foo",
					StagingUploaderEntrypoint:   "bar",
					StagingExecutorEntrypoint:   "baz",
					RuntimeEntrypoint:           "42",
				})

				Expect(gracefulInjector.Options.FailGracePeriod).To(Equal("12"))
				Expect(gracefulInjector.Options.SuccessGracePeriod).To(Equal("90"))
				Expect(gracefulInjector.Options.StagingDownloaderEntrypoint).To(Equal("foo"))
				Expect(gracefulInjector.Options.StagingUploaderEntrypoint).To(Equal("bar"))
				Expect(gracefulInjector.Options.StagingExecutorEntrypoint).To(Equal("baz"))
				Expect(gracefulInjector.Options.RuntimeEntrypoint).To(Equal("42"))
			})
		})
	})

	Describe("GracePeriod Injector", func() {
		Context("when handling a Eirini runtime app", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "opi",
							},
						},
					},
				}
			})
			It("Injects a grace period", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(addOpiPatch))
			})
		})

		Context("when a non-opi pod is handled", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "foo",
							},
						},
					},
				}
			})
			It("Does not inject a grace period", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(""))
			})
		})

		Context("when a staging pod is handled", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						Containers: []corev1.Container{
							{
								Name: "opi-task-uploader",
							},
						},
					},
				}
			})
			It("Does inject a grace period if the container is opi-task-uploader", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(addUploaderPatch))
			})
		})

		Context("when a pod is handled", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "opi-task-uploader",
							},
						},
					},
				}
			})
			It("Does not inject a grace period if the initcontainer doesn't have the correct name", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(""))
			})
		})

		Context("when a pod is handled", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "opi-task-downloader",
							},
						},
					},
				}
			})
			It("Does inject a grace period if the initcontainer have the opi-task-downloader name", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(addDownloaderPatch))
			})
		})

		Context("when a staging pod is handled", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "opi-task-executor",
							},
						},
					},
				}
			})
			It("Does inject a grace period if the initcontainer if have the opi-task-executor name", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(addExecutorPatch))
			})
		})

		Context("when a nil pod is passed by", func() {
			BeforeEach(func() {
				pod = nil
			})

			It("Doesn't return any patch", func() {
				Expect(decodePatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))).To(Equal(""))
			})
		})

		Context("when a (full) staging pod is handled", func() {
			BeforeEach(func() {
				pod = &corev1.Pod{
					Spec: corev1.PodSpec{
						InitContainers: []corev1.Container{
							{
								Name: "opi-task-executor",
							},
							{
								Name: "opi-task-downloader",
							},
						},
						Containers: []corev1.Container{
							{
								Name: "opi-task-uploader",
							},
						},
					},
				}
			})

			It("Does inject a grace period in all Containers and InitContainers", func() {
				patches := jsonifyPatches(gracefulInjector.Handle(context.TODO(), eiriniManager, pod, request))
				Expect(patches).To(ContainElement(stagingFullPatchExecutor))
				Expect(patches).To(ContainElement(stagingFullPatchUploader))
				Expect(patches).To(ContainElement(stagingFullPatchDownloader))
				Expect(len(patches)).To(Equal(3))
			})
		})
	})
})
