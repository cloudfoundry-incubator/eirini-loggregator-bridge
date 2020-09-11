package podwatcher

import (
	"context"
	"fmt"
	"time"

	eirinix "code.cloudfoundry.org/eirinix"
	log "code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func NewLogReconciler(pw *PodWatcher, gracefulStartTime string) *logReconciler {
	if gracefulStartTime == "" {
		gracefulStartTime = "10"
	}
	return &logReconciler{pw: pw, gracefulStartTime: gracefulStartTime}
}

type logReconciler struct {
	mgr               eirinix.Manager
	pw                *PodWatcher
	gracefulStartTime string
}

func (r *logReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Injects PreStart hook for gracefulStarts:
	// In case an application is too fast to fail, we might miss logs as Kubernetes will terminate the container
	// before the watcher had occasion to attach to stream any logs.
	// - Finalizers let the pod terminates, not giving any chance to get logs from terminated container.
	// - PostStop hooks doesn't guarantee execution in case of containers terminated by failures (e.g. an invalid app was pushed in Eirini)
	// - With the PreStart instead, we give chance to the watcher to hook to the container, without modifying the standard execution flow of the app
	gracefulStart := &v1.Lifecycle{PostStart: &v1.Handler{Exec: &v1.ExecAction{Command: []string{
		"/bin/sh",
		"-c", "sleep " + r.gracefulStartTime,
	}}}}

	ctx := log.NewContextWithRecorder(r.mgr.GetContext(), "loggregator-bridge-reconciler", r.mgr.GetKubeManager().GetEventRecorderFor("loggregator-bridge"))
	pod := &corev1.Pod{}

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Info(ctx, "Reconciling pod ", request.NamespacedName)
	if err := r.mgr.GetKubeManager().GetClient().Get(ctx, request.NamespacedName, pod); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	for i, _ := range pod.Spec.InitContainers {
		c := &pod.Spec.InitContainers[i] //	if c.Lifecycle == nil {
		c.Lifecycle = gracefulStart
		//		}
	}
	for i, _ := range pod.Spec.Containers {
		c := &pod.Spec.Containers[i]
		//	if c.Lifecycle == nil {
		c.Lifecycle = gracefulStart
		//	}
	}

	if err := r.mgr.GetKubeManager().GetClient().Update(ctx, pod); err != nil {
		log.WithEvent(pod, "UpdateError").Errorf(ctx, "Failed to update pod gracefulStart '%s/%s' (%v): %s", pod.Namespace, pod.Name, pod.ResourceVersion, err)
		return reconcile.Result{Requeue: true}, nil
	}
	fmt.Println("Adding gracefulstart to", pod)
	log.WithEvent(pod, "Info").Infof(ctx, "Updated pod gracefulStart '%s/%s' (%v)", pod.Namespace, pod.Name, pod.ResourceVersion)

	return reconcile.Result{}, nil
}

func (r *logReconciler) Register(m eirinix.Manager) error {
	r.mgr = m

	nsPred := predicate.Funcs{
		CreateFunc: func(e event.CreateEvent) bool {
			return m.GetManagerOptions().Namespace == e.Meta.GetNamespace()
		},
		DeleteFunc: func(e event.DeleteEvent) bool {
			return m.GetManagerOptions().Namespace == e.Meta.GetNamespace()
		},
		GenericFunc: func(e event.GenericEvent) bool {
			return m.GetManagerOptions().Namespace == e.Meta.GetNamespace()

		},
		UpdateFunc: func(e event.UpdateEvent) bool {
			return m.GetManagerOptions().Namespace == e.MetaNew.GetNamespace()

		},
	}

	c, err := controller.New("log-controller", m.GetKubeManager(), controller.Options{
		Reconciler: r,
	})
	if err != nil {
		return errors.Wrap(err, "Adding log controller to manager failed.")
	}
	// watch pods, trigger if one pod is created
	p := predicate.Funcs{
		CreateFunc:  func(e event.CreateEvent) bool { return true },
		DeleteFunc:  func(e event.DeleteEvent) bool { return true },
		GenericFunc: func(e event.GenericEvent) bool { return true },
		UpdateFunc:  func(e event.UpdateEvent) bool { return true },
	}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestsFromMapFunc{
		ToRequests: handler.ToRequestsFunc(func(a handler.MapObject) []reconcile.Request {
			pod := a.Object.(*corev1.Pod)

			result := []reconcile.Request{}
			result = append(result, reconcile.Request{
				NamespacedName: types.NamespacedName{
					Name:      pod.Name,
					Namespace: pod.Namespace,
				}})
			return result
		}),
	}, nsPred, p)
	if err != nil {
		return errors.Wrapf(err, "Watching pods failed in log controller.")
	}

	config, err := r.mgr.GetKubeConnection()
	if err != nil {
		return errors.Wrapf(err, "getting kubernetes connection.")
	}
	r.pw.Containers.KubeConfig = config
	r.pw.Containers.Context = r.mgr.GetContext()
	r.pw.Containers.LoggregatorOptions = r.pw.Config.GetLoggregatorOptions()
	return nil
}
