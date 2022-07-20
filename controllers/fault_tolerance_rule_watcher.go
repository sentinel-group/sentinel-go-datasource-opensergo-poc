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
	datasourcev1 "github.com/sentinel-group/sentinel-go-datasource-opensergo-poc/api/v1alpha1"
	k8sApiError "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"net/http"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// FaultToleranceRuleReconciler reconciles a FaultToleranceRule object
type FaultToleranceRuleReconciler struct {
	client.Client
	logger logr.Logger
	scheme *runtime.Scheme

	namespace string
	appName   string

	ruleAggregator *FaultToleranceRuleAggregator
}

// +kubebuilder:rbac:groups=fault-tolerance.opensergo.io,resources=FaultToleranceRule,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=fault-tolerance.opensergo.io,resources=FaultToleranceRule/status,verbs=get;update;patch

func (r *FaultToleranceRuleReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	if req.Namespace != r.namespace {
		// Ignore unmatched namespace
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, nil
	}
	log := r.logger.WithValues("expectedNamespace", r.namespace, "expectedAppName", r.appName, "req", req.String())

	// your logic here
	ftRuleCR := &datasourcev1.FaultToleranceRule{}
	if err := r.Get(ctx, req.NamespacedName, ftRuleCR); err != nil {
		k8sApiErr, ok := err.(*k8sApiError.StatusError)
		if !ok {
			log.Error(err, "Failed to get OpenSergo FaultToleranceRule")
			return ctrl.Result{
				Requeue:      false,
				RequeueAfter: 0,
			}, nil
		}
		if k8sApiErr.Status().Code != http.StatusNotFound {
			log.Error(err, "Failed to get OpenSergo FaultToleranceRule")
			return ctrl.Result{
				Requeue:      false,
				RequeueAfter: 0,
			}, nil
		}

		// cr had been deleted
		ftRuleCR = nil
	}

	if ftRuleCR != nil {
		app, exists := ftRuleCR.Labels["app"]
		if !exists || app != r.appName {
			if _, prevContains := r.ruleAggregator.getRule(ftRuleCR.Name); prevContains {
				log.Info("OpenSergo FaultToleranceRule will be deleted because app label has been changed", "newApp", app)
				ftRuleCR = nil
			} else {
				// Ignore unmatched app label
				return ctrl.Result{
					Requeue:      false,
					RequeueAfter: 0,
				}, nil
			}
		} else {
			log.Info("OpenSergo FaultToleranceRule received", "rule", ftRuleCR.Spec)
		}
	} else {
		log.Info("OpenSergo FaultToleranceRule will be deleted")
	}

	_, err := r.ruleAggregator.updateRule(req.Name, ftRuleCR)
	if err != nil {
		log.Error(err, "Fail to updateRule to FaultToleranceRuleAggregator")
		return ctrl.Result{
			Requeue:      false,
			RequeueAfter: 0,
		}, nil
	}

	return ctrl.Result{}, nil
}

func (r *FaultToleranceRuleReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).For(&datasourcev1.FaultToleranceRule{}).Complete(r)
}

func NewFaultToleranceRuleReconciler(crdManager ctrl.Manager, namespace string, app string, ruleAggregator *FaultToleranceRuleAggregator) *FaultToleranceRuleReconciler {
	return &FaultToleranceRuleReconciler{
		Client:         crdManager.GetClient(),
		logger:         ctrl.Log.WithName("controllers").WithName("fault-tolerance.opensergo.io/FaultToleranceRule"),
		scheme:         crdManager.GetScheme(),
		namespace:      namespace,
		appName:        app,
		ruleAggregator: ruleAggregator,
	}
}
