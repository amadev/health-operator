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
	"fmt"
	"github.com/go-logr/logr"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"

	commonv1alpha1 "github.com/amadev/health-operator/api/v1alpha1"
)

// HealthReconciler reconciles a Health object
type HealthReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=common.amadev.ru,resources=healths,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=common.amadev.ru,resources=healths/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=apps,resources=daemonsets,verbs=get;list;watch;create;update;patch;delete

func (r *HealthReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("health", req.NamespacedName)

	log.Info("Got reconcile request")

	health := &commonv1alpha1.Health{}
	err := r.Get(ctx, types.NamespacedName{Name: "health", Namespace: req.Namespace}, health)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			log.Info("Health resource not found. Ignoring since object must be deleted")
			return ctrl.Result{}, nil
		}
		// Error reading the object - requeue the request.
		log.Error(err, "Failed to get Health")
		return ctrl.Result{}, err
	}

	found := &appsv1.Deployment{}
	err = r.Get(ctx, types.NamespacedName{Name: req.Name, Namespace: req.Namespace}, found)
	if err != nil {
		log.Error(err, "Failed to get Deployment")
		return ctrl.Result{}, err
	}

	app, app_ok := found.ObjectMeta.Labels["application"]
	component, component_ok := found.ObjectMeta.Labels["component"]
	alt_name := strings.Split(req.Name, "-")
	log.Info("Identification", "app", app, "component", component, "alt_name", alt_name)
	if app_ok == false {
		app = alt_name[0]
	}
	if component_ok == false {
		if len(alt_name) > 1 {
			component = alt_name[1]
		} else {
			component = "default"
		}
	}

	available := false
	progressing := false
	for i := 0; i < len(found.Status.Conditions); i++ {
		c := found.Status.Conditions[i]
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

	log.Info("Status", "available", available, "progressing", progressing, "status", status)

	patch := []byte(fmt.Sprintf(`{"status":{"%s": {"%s": {"status": "%s", "generation": %d}}}}`, app, component, status, found.Generation))
	err = r.Status().Patch(ctx, health, client.RawPatch(types.MergePatchType, patch))
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
