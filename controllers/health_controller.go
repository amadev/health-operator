/*


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

package controllers

import (
	"context"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	commonv1alpha1 "github.com/amadev/health-operator/api/v1alpha1"
)

type HealthReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

func (r *HealthReconciler) calculateStatus(obj *appsv1.Deployment) string {
	available := false
	progressing := false
	for i := 0; i < len(obj.Status.Conditions); i++ {
		c := obj.Status.Conditions[i]
		if c.Status == "True" && c.Type == "Available" {
			available = true
		}
		if c.Status == "True" && c.Type == "Progressing" && c.Reason == "NewReplicaSetAvailable" {
			progressing = true
		}
	}
	status := "notready"
	if available && progressing {
		status = "ready"
	}
	return status

}

// +kubebuilder:rbac:groups=common.amadev.ru,resources=healths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.amadev.ru,resources=healths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

func (r *HealthReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("health", req.NamespacedName)
	found := &appsv1.Deployment{}
	log.Info("Got reconcile request")

	health := &commonv1alpha1.Health{}
	err := r.Get(ctx, types.NamespacedName{Name: "health", Namespace: req.Namespace}, health)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Health resource not found. Until health object is present in the namespace, summary will not be created")
			return ctrl.Result{}, nil
		}
		log.Error(err, "Failed to get Health")
		return ctrl.Result{}, err
	}

	err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, found)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Info("Object was deleted. Finalazer is not used. Skipping that case")
			return ctrl.Result{}, nil
		}

		log.Error(err, "Failed to get an object")
		return ctrl.Result{}, err
	}

	app, component := getIdentity(found.ObjectMeta)
	log.Info("Identification", "app", app, "component", component)

	status := r.calculateStatus(found)

	log.Info("Status", "status", status)

	patch := getPatch(app, component, status, found.Generation)

	err = r.Status().Patch(
		ctx,
		health,
		client.RawPatch(types.MergePatchType, patch))

	if err != nil {
		log.Error(err, "Failed to update Health status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func (r *HealthReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&appsv1.Deployment{}).
		Complete(r)
}
