package controllers

import (
	"context"

	log "github.com/sirupsen/logrus"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/event"

	argoprojiov1alpha1 "github.com/argoproj-labs/applicationset/api/v1alpha1"
	"github.com/argoproj-labs/applicationset/pkg/generators"
)

// clusterSecretEventHandler is used when watching Secrets to check if they are ArgoCD Cluster Secrets, and if so
// requeue any related ApplicationSets.
type clusterSecretEventHandler struct {
	//handler.EnqueueRequestForOwner
	Log    log.FieldLogger
	Client client.Client
}

func (h *clusterSecretEventHandler) Create(e event.CreateEvent, q workqueue.RateLimitingInterface) {
	h.queueRelatedAppGenerators(q, e.Meta)
}

func (h *clusterSecretEventHandler) Update(e event.UpdateEvent, q workqueue.RateLimitingInterface) {
	h.queueRelatedAppGenerators(q, e.MetaNew)
}

func (h *clusterSecretEventHandler) Delete(e event.DeleteEvent, q workqueue.RateLimitingInterface) {
	h.queueRelatedAppGenerators(q, e.Meta)
}

func (h *clusterSecretEventHandler) Generic(e event.GenericEvent, q workqueue.RateLimitingInterface) {
	h.queueRelatedAppGenerators(q, e.Meta)
}

func (h *clusterSecretEventHandler) queueRelatedAppGenerators(q workqueue.RateLimitingInterface, meta metav1.Object) {
	// Check for label, lookup all ApplicationSets that might match the cluster, queue them all
	if meta.GetLabels()[generators.ArgoCDSecretTypeLabel] != generators.ArgoCDSecretTypeCluster {
		return
	}

	h.Log.WithFields(log.Fields{
		"namespace": meta.GetNamespace(),
		"name":      meta.GetName(),
	}).Info("processing event for cluster secret")

	appSetList := &argoprojiov1alpha1.ApplicationSetList{}
	err := h.Client.List(context.Background(), appSetList)
	if err != nil {
		h.Log.WithError(err).Error("unable to list ApplicationSets")
		return
	}
	h.Log.WithField("count", len(appSetList.Items)).Info("listed ApplicationSets")
	for _, appSet := range appSetList.Items {
		foundClusterGenerator := false
		for _, generator := range appSet.Spec.Generators {
			if generator.Clusters != nil {
				foundClusterGenerator = true
				break
			}
		}
		if foundClusterGenerator {
			// TODO: only queue the AppGenerator if the labels match this cluster
			req := ctrl.Request{NamespacedName: types.NamespacedName{Namespace: appSet.Namespace, Name: appSet.Name}}
			q.Add(req)
		}
	}
}

