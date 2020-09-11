package podwatcher

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"runtime"

	eirinix "code.cloudfoundry.org/eirinix"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// Extension changes pod definitions
type Extension struct {
	Logger            *zap.SugaredLogger
	gracefulStartTime string
}

// NewGracefulStartTime returns the podwatcher extension
func NewGracefulStartTime(gracefulStartTime string) eirinix.Extension {
	return &Extension{gracefulStartTime: gracefulStartTime}
}

// Handle manages volume claims for ExtendedStatefulSet pods
func (ext *Extension) Handle(ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req admission.Request) admission.Response {
	// Injects PreStart hook for gracefulStarts:
	// In case an application is too fast to fail, we might miss logs as Kubernetes will terminate the container
	// before the watcher had occasion to attach to stream any logs.
	// - Finalizers let the pod terminates, not giving any chance to get logs from terminated container.
	// - PostStop hooks doesn't guarantee execution in case of containers terminated by failures (e.g. an invalid app was pushed in Eirini)
	// - With the PreStart instead, we give chance to the watcher to hook to the container, without modifying the standard execution flow of the app
	gracefulStart := &v1.Lifecycle{PostStart: &v1.Handler{Exec: &v1.ExecAction{Command: []string{
		"/bin/sh",
		"-c", "sleep " + ext.gracefulStartTime,
	}}}}

	if pod == nil {
		return admission.Errored(http.StatusBadRequest, errors.New("No pod could be decoded from the request"))
	}

	_, file, _, _ := runtime.Caller(0)
	log := eiriniManager.GetLogger().Named(file)

	ext.Logger = log
	podCopy := pod.DeepCopy()
	log.Debugf("Handling webhook request for POD: %s (%s)", podCopy.Name, podCopy.Namespace)
	// Init containers does not have poststart
	//	for i, _ := range podCopy.Spec.InitContainers {
	//	c := &podCopy.Spec.InitContainers[i]
	//	if c.Lifecycle == nil {
	//	c.Lifecycle = gracefulStart
	//		}
	//}
	for i, _ := range podCopy.Spec.Containers {
		c := &podCopy.Spec.Containers[i]

		//	if c.Lifecycle == nil {
		c.Lifecycle = gracefulStart
		//	}
	}
	fmt.Println(podCopy)

	return eiriniManager.PatchFromPod(req, podCopy)
}
