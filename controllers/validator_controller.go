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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/intstr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	terrav1alpha1 "github.com/terra-rebels/terra-operator/api/v1alpha1"
)

type ValidatorReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

func (r *ValidatorReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&terrav1alpha1.Validator{}).
		Owns(&terrav1alpha1.TerradNode{}).
		Owns(&terrav1alpha1.OracleFeeder{}).
		Owns(&corev1.Service{}).
		Complete(r)
}

//+kubebuilder:rbac:groups=terra.terra-rebels.org,resources=validators,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=terra.terra-rebels.org,resources=validators/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=terra.terra-rebels.org,resources=validators/finalizers,verbs=update
func (r *ValidatorReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling Validator object")

	validator := &terrav1alpha1.Validator{}
	err := r.Client.Get(context.TODO(), req.NamespacedName, validator)

	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	terradNode := newTerradNodeForValidator(validator)

	if err := controllerutil.SetControllerReference(validator, terradNode, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundTerrad := &terrav1alpha1.TerradNode{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: terradNode.Name, Namespace: terradNode.Namespace}, foundTerrad)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new TerradNode", "TerradNode.Namespace", terradNode.Namespace, "TerradNode.Name", terradNode.Name)

		err = r.Client.Create(context.TODO(), terradNode)

		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	oracleFeeder := newOracleFeederForValidator(validator)

	if err := controllerutil.SetControllerReference(validator, oracleFeeder, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundOracleFeeder := &terrav1alpha1.OracleFeeder{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: oracleFeeder.Name, Namespace: oracleFeeder.Namespace}, foundOracleFeeder)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new OracleFeeder", "OracleFeeder.Namespace", oracleFeeder.Namespace, "OracleFeeder.Name", oracleFeeder.Name)

		err = r.Client.Create(context.TODO(), oracleFeeder)

		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	if validator.Spec.IsPublic {
		service := newServiceForValidator(validator)

		if err := controllerutil.SetControllerReference(validator, service, r.Scheme); err != nil {
			return ctrl.Result{}, err
		}

		foundService := &corev1.Service{}
		err = r.Client.Get(context.TODO(), types.NamespacedName{Name: service.Name, Namespace: service.Namespace}, foundService)

		if err != nil && errors.IsNotFound(err) {
			logger.Info("Creating a new Service", "Service.Namespace", service.Namespace, "Service.Name", service.Name)

			err = r.Client.Create(context.TODO(), service)

			if err != nil {
				return ctrl.Result{}, err
			}

			return ctrl.Result{}, nil
		} else if err != nil {
			return ctrl.Result{}, err
		}
	}

	return ctrl.Result{}, nil
}

func newOracleFeederForValidator(cr *terrav1alpha1.Validator) *terrav1alpha1.OracleFeeder {
	labels := map[string]string{
		"app": cr.Name,
	}

	oracleFeeder := &terrav1alpha1.OracleFeeder{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-oraclefeeder",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: terrav1alpha1.OracleFeederSpec{
			ChainId:   cr.Spec.ChainId,
			NodeImage: cr.Spec.OracleFeederNodeImage,
		},
	}

	return oracleFeeder
}

func newTerradNodeForValidator(cr *terrav1alpha1.Validator) *terrav1alpha1.TerradNode {
	labels := map[string]string{
		"app": cr.Name,
	}

	//TODO: Replace this with a more complex bash script that will bootstrap a validator from the "ground-up"
	postStartCommand := fmt.Sprintf(`terrad tx staking create-validator 
		--pubkey=$(terrad tendermint show-validator) 		
		--chain-id=%s
		--moniker="%s" 
		--from=%s
		--amount=%s
		--commission-rate="%s" 
		--commission-max-rate="%s" 
		--commission-max-change-rate="%s" 
		--min-self-delegation="%s"
		--gas auto
		--node tcp://127.0.0.1:26647`,
		cr.Spec.ChainId,
		cr.Name,
		cr.Spec.FromKeyName,
		cr.Spec.InitialSelfBondAmount,
		cr.Spec.InitialCommissionRate,
		cr.Spec.MaximumCommission,
		cr.Spec.CommissionChangeRate,
		cr.Spec.MinimumSelfBondAmount)

	terrad := &terrav1alpha1.TerradNode{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-terrad",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: terrav1alpha1.TerradNodeSpec{
			NodeImage:  cr.Spec.TerradNodeImage,
			IsFullNode: true,
			DataVolume: cr.Spec.DataVolume,
			PostStartCommand: []string{
				postStartCommand,
			},
		},
	}

	return terrad
}

func newServiceForValidator(cr *terrav1alpha1.Validator) *corev1.Service {
	labels := map[string]string{
		"app": cr.Name,
	}

	selector := map[string]string{
		"app": cr.Name + "-terrad",
	}

	ports := []corev1.ServicePort{
		{
			Name:       "p2p",
			Port:       26656,
			TargetPort: intstr.FromString("p2p"),
		},
		{
			Name:       "rpc",
			Port:       26657,
			TargetPort: intstr.FromString("rpc"),
		},
		{
			Name:       "lcd",
			Port:       1317,
			TargetPort: intstr.FromString("lcd"),
		},
	}

	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-service",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Ports:    ports,
			Selector: selector,
		},
	}
}
