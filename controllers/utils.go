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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

func getIdentity(meta metav1.ObjectMeta) (app, component string) {
	app, app_ok := meta.Labels["application"]
	component, component_ok := meta.Labels["component"]
	alt_name := strings.Split(meta.Name, "-")
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
	return app, component
}

func updateHealth(r interface{}, health runtime.Object, app string, component string, status string, generation int64) (ctrl.Result, error) {

	log := r.(HealthReconciler).Log
	ctx := context.Background()
	patch := []byte(fmt.Sprintf(`{"status":{"%s": {"%s": {"status": "%s", "generation": %d}}}}`, app, component, status, generation))
	err := r.(HealthReconciler).Status().Patch(ctx, health, client.RawPatch(types.MergePatchType, patch))
	if err != nil {
		log.Error(err, "Failed to update Health status")
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}
