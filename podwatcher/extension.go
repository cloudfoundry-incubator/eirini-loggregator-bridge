package podwatcher

import (
	"context"
	"errors"
	"net/http"
	"runtime"

	eirinix "code.cloudfoundry.org/eirinix"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

// https://github.com/cloudfoundry-incubator/eirini-staging/tree/master/image
const (
	DefaultStagingExecutorEntrypoint   = "/packs/executor"
	DefaultStagingDownloaderEntrypoint = "/packs/downloader"
	DefaultStagingUploaderEntrypoint   = "/packs/uploader"
	DefaultRuntimeEntrypoint           = "/lifecycle/launch"
	DefaultFailGracePeriod             = "5"
	DefaultSuccessGracePeriod          = "5"
)

// GraceOptions lets customize the graceful periods and
// the entrypoint of the images which are mutated to inject
// the grace period logic
type GraceOptions struct {
	FailGracePeriod, SuccessGracePeriod string

	StagingDownloaderEntrypoint string
	StagingExecutorEntrypoint   string
	StagingUploaderEntrypoint   string
	RuntimeEntrypoint           string
}

// Extension changes pod definitions
type Extension struct {
	Logger  *zap.SugaredLogger
	Options GraceOptions
}

// NewGracePeriodInjector returns the podwatcher extension which injects a grace Period on Eirini generated pods
func NewGracePeriodInjector(opts *GraceOptions) *Extension {
	if len(opts.StagingExecutorEntrypoint) == 0 {
		opts.StagingExecutorEntrypoint = DefaultStagingExecutorEntrypoint
	}

	if len(opts.StagingDownloaderEntrypoint) == 0 {
		opts.StagingDownloaderEntrypoint = DefaultStagingDownloaderEntrypoint
	}

	if len(opts.StagingUploaderEntrypoint) == 0 {
		opts.StagingUploaderEntrypoint = DefaultStagingUploaderEntrypoint
	}

	if len(opts.RuntimeEntrypoint) == 0 {
		opts.RuntimeEntrypoint = DefaultRuntimeEntrypoint
	}

	if len(opts.FailGracePeriod) == 0 {
		opts.FailGracePeriod = DefaultFailGracePeriod
	}

	if len(opts.SuccessGracePeriod) == 0 {
		opts.SuccessGracePeriod = DefaultSuccessGracePeriod
	}

	return &Extension{Options: *opts}
}

// Handle injects gracefulPeriod in opi containers:
// In case an application is too fast to fail, we might miss logs as Kubernetes will terminate the container
// before the watcher had occasion to attach to stream any logs.
// - Finalizers let the pod terminates, not giving any chance to get logs from terminated container.
// - PostStop/PreStart hooks doesn't guarantee execution in case of containers terminated by failures (e.g. an invalid app was pushed in Eirini), if causing crashloopbackoff
//   hooks aren't executed correctly.
func (ext *Extension) Handle(ctx context.Context, eiriniManager eirinix.Manager, pod *corev1.Pod, req admission.Request) admission.Response {

	if pod == nil {
		return admission.Errored(http.StatusBadRequest, errors.New("No pod could be decoded from the request"))
	}

	_, file, _, _ := runtime.Caller(0)
	log := eiriniManager.GetLogger().Named(file)

	ext.Logger = log
	podCopy := pod.DeepCopy()
	log.Debugf("Handling webhook request for POD: %s (%s)", podCopy.Name, podCopy.Namespace)

	for i := range podCopy.Spec.InitContainers {
		c := &podCopy.Spec.InitContainers[i]
		switch c.Name {
		case "opi-task-downloader":
			c.Command = []string{"/bin/sh", "-c", "( " + ext.Options.StagingDownloaderEntrypoint + " && sleep " + ext.Options.SuccessGracePeriod + " ) || sleep " + ext.Options.FailGracePeriod + ""}
		case "opi-task-executor":
			c.Command = []string{"/bin/sh", "-c", "( " + ext.Options.StagingExecutorEntrypoint + " && sleep " + ext.Options.SuccessGracePeriod + " ) || sleep " + ext.Options.FailGracePeriod + ""}
		}
	}

	for i := range podCopy.Spec.Containers {
		c := &podCopy.Spec.Containers[i]
		switch c.Name {
		case "opi":
			c.Command = []string{"dumb-init", "--", "/bin/sh", "-c", "(  " + ext.Options.RuntimeEntrypoint + " && sleep " + ext.Options.SuccessGracePeriod + " ) || sleep " + ext.Options.FailGracePeriod}
		case "opi-task-uploader":
			c.Command = []string{"/bin/sh", "-c", "( " + ext.Options.StagingUploaderEntrypoint + " && sleep " + ext.Options.SuccessGracePeriod + " ) || sleep " + ext.Options.FailGracePeriod}
		}
	}

	return eiriniManager.PatchFromPod(req, podCopy)
}
