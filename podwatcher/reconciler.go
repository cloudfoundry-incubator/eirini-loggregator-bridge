package podwatcher

import (
	"context"
	"fmt"
	"time"

	eirinix "code.cloudfoundry.org/eirinix"
	log "code.cloudfoundry.org/quarks-utils/pkg/ctxlog"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"
)

func NewLogReconciler(pw *PodWatcher) *logReconciler {
	return &logReconciler{pw: pw}
}

type logReconciler struct {
	mgr eirinix.Manager
	pw  *PodWatcher
}

func (r *logReconciler) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	ctx := log.NewContextWithRecorder(r.mgr.GetContext(), "loggregator-bridge-reconciler", r.mgr.GetKubeManager().GetEventRecorderFor("loggregator-bridge"))
	pod := &corev1.Pod{}

	// Set the ctx to be Background, as the top-level context for incoming requests.
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	log.Info(ctx, "Reconciling pod ", request.NamespacedName)
	if err := r.mgr.GetKubeManager().GetClient().Get(ctx, request.NamespacedName, pod); err != nil {
		return reconcile.Result{Requeue: true}, err
	}

	// name of our custom finalizer
	myFinalizerName := "eirinix-finalizers.io/loggregator-bridge"

	// examine DeletionTimestamp to determine if object is under deletion
	if pod.ObjectMeta.DeletionTimestamp.IsZero() {
		// The object is not being deleted, so if it does not have our finalizer,
		// then lets add the finalizer and update the object. This is equivalent
		// registering our finalizer.
		if !containsString(pod.ObjectMeta.Finalizers, myFinalizerName) {
			pod.ObjectMeta.Finalizers = append(pod.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.mgr.GetKubeManager().GetClient().Update(ctx, pod); err != nil {
				log.WithEvent(pod, "UpdateError").Errorf(ctx, "Failed to update pod finalizer '%s/%s' (%v): %s", pod.Namespace, pod.Name, pod.ResourceVersion, err)
				return reconcile.Result{}, nil
			}
			fmt.Println("Adding finalizer to", pod)
			log.WithEvent(pod, "Info").Infof(ctx, "Updated pod finalizer '%s/%s' (%v)", pod.Namespace, pod.Name, pod.ResourceVersion)

		}
	} else {
		// The object is being deleted
		if containsString(pod.ObjectMeta.Finalizers, myFinalizerName) {
			// our finalizer is present, so lets handle any external dependency
			if err := r.ensureLogsAreStreamed(pod); err != nil {
				// if fail to delete the external dependency here, return with error
				// so that it can be retried
				return reconcile.Result{}, nil
			}
			fmt.Println("Removing finalizer from", pod)

			// remove our finalizer from the list and update it.
			pod.ObjectMeta.Finalizers = removeString(pod.ObjectMeta.Finalizers, myFinalizerName)
			if err := r.mgr.GetKubeManager().GetClient().Update(ctx, pod); err != nil {
				log.WithEvent(pod, "UpdateError").Errorf(ctx, "Failed to update pod finalizer '%s/%s' (%v): %s", pod.Namespace, pod.Name, pod.ResourceVersion, err)
				return reconcile.Result{}, nil
			}
			//		if err := r.dropLoggingFromPod(pod); err != nil {
			//		log.WithEvent(pod, "UpdateError").Errorf(ctx, "Failed to update remove pod from logging processes '%s/%s' (%v): %s", pod.Namespace, pod.Name, pod.ResourceVersion, err)
			//		return reconcile.Result{}, nil
			//	}
			log.WithEvent(pod, "Info").Infof(ctx, "Removed pod finalizer '%s/%s' (%v)", pod.Namespace, pod.Name, pod.ResourceVersion)
		}

		r.pw.Containers.EnsurePodStatus(pod)
		// Stop reconciliation as the item is being deleted
		return reconcile.Result{}, nil
	}

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

func (r *logReconciler) ensureLogsAreStreamed(pod *corev1.Pod) error {
	podContainers := ExtractContainersFromPod(pod)
	for _, c := range podContainers {
		if _, ok := r.pw.Containers.GetContainer(c.UID); !ok {
			fmt.Println("Log not streamed yet, not removing the finalizer from", c)
			return errors.New("Logs not streamed yet")
		}
	}
	return nil
}

func (r *logReconciler) dropLoggingFromPod(pod *corev1.Pod) error {
	podContainers := ExtractContainersFromPod(pod)
	for _, c := range podContainers {
		if err := r.pw.Containers.RemoveContainer(c.UID); err != nil {
			return err
		}
	}
	return nil
}

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func removeString(slice []string, s string) (result []string) {
	for _, item := range slice {
		if item == s {
			continue
		}
		result = append(result, item)
	}
	return
}
