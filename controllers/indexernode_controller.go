/*
Copyright 2022.

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

	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	terrav1alpha1 "github.com/terra-rebels/terra-operator/api/v1alpha1"
)

// IndexerNodeReconciler reconciles a IndexerNode object
type IndexerNodeReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// SetupWithManager sets up the controller with the Manager.
func (r *IndexerNodeReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&terrav1alpha1.IndexerNode{}).
		Complete(r)
}

// +kubebuilder:rbac:groups=terra.terra-rebels.org,resources=indexernodes,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=terra.terra-rebels.org,resources=indexernodes/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=terra.terra-rebels.org,resources=indexernodes/finalizers,verbs=update
func (r *IndexerNodeReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	_ = log.FromContext(ctx)

	// TODO(user): your logic here

	return ctrl.Result{}, nil
}
