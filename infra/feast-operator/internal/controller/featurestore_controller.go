/*
Copyright 2024 Feast Community.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"reflect"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
)

// Constants for requeue
const (
	RequeueDelaySuccess = 10 * time.Second
	RequeueDelayError   = 5 * time.Second
)

// FeatureStoreReconciler reconciles a FeatureStore object
type FeatureStoreReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=feast.dev,resources=featurestores,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=feast.dev,resources=featurestores/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=feast.dev,resources=featurestores/finalizers,verbs=update
//+kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;create;update;watch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;create;update;watch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.17.3/pkg/reconcile
func (r *FeatureStoreReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	cr := &feastdevv1alpha1.FeatureStore{}
	err := r.Get(ctx, req.NamespacedName, cr)
	if err != nil {
		if apierrors.IsNotFound(err) {
			// CR deleted since request queued, child objects getting GC'd, no requeue
			logger.V(1).Info("FeatureStore CR not found, has been deleted")
			return ctrl.Result{}, nil
		}
		// error fetching FeatureStore instance, requeue and try again
		logger.Error(err, "Error in Get of FeatureStore CR")
		return ctrl.Result{}, err
	}

	nextStatus := cr.Status.DeepCopy()
	nextStatus.Applied = cr.Spec

	if cr.DeletionTimestamp == nil {
		setStatusCondition(&nextStatus.Conditions, feastdevv1alpha1.ReadyType, metav1.ConditionTrue, feastdevv1alpha1.ReadyReason, feastdevv1alpha1.ReadyMessage)
		logger.Info("FeatureStore installation complete")
	}

	return r.updateStatus(cr, nextStatus)
}

// SetupWithManager sets up the controller with the Manager.
func (r *FeatureStoreReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&feastdevv1alpha1.FeatureStore{}).
		Owns(&appsv1.Deployment{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

// setStatusCondition sets the given condition with the given status, reason and message on a resource.
func setStatusCondition(Conditions *[]metav1.Condition, conditionType string, status metav1.ConditionStatus, reason, message string) {
	newCondition := metav1.Condition{
		Type:    conditionType,
		Status:  status,
		Reason:  reason,
		Message: message,
	}

	apimeta.SetStatusCondition(Conditions, newCondition)
}

func (r *FeatureStoreReconciler) updateStatus(cr *feastdevv1alpha1.FeatureStore, nextStatus *feastdevv1alpha1.FeatureStoreStatus) (ctrl.Result, error) {
	if !reflect.DeepEqual(&cr.Status, nextStatus) {
		nextStatus.DeepCopyInto(&cr.Status)
		err := r.Client.Status().Update(context.Background(), cr)
		if err != nil {
			return ctrl.Result{
				Requeue:      true,
				RequeueAfter: RequeueDelayError,
			}, err
		}
	}

	return ctrl.Result{}, nil
}
