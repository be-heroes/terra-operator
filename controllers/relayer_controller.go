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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	terrav1alpha1 "github.com/terra-rebels/terra-operator/api/v1alpha1"
)

// RelayerReconciler reconciles a Relayer object
type RelayerReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=terra.terra-rebels.org,resources=relayers,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=terra.terra-rebels.org,resources=relayers/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=terra.terra-rebels.org,resources=relayers/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the Relayer object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *RelayerReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	logger := log.FromContext(ctx)

	logger.Info("Reconciling Relayer object")

	relayer := &terrav1alpha1.Relayer{}
	err := r.Client.Get(ctx, req.NamespacedName, relayer)

	if err != nil {
		if errors.IsNotFound(err) {
			return ctrl.Result{}, nil
		}

		return ctrl.Result{}, err
	}

	pod := newPodForRelayer(relayer)

	if err := controllerutil.SetControllerReference(relayer, pod, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}

	foundPod := &corev1.Pod{}
	err = r.Client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, foundPod)

	if err != nil && errors.IsNotFound(err) {
		logger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)

		err = r.Client.Create(context.TODO(), pod)

		if err != nil {
			return ctrl.Result{}, err
		}

		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *RelayerReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&terrav1alpha1.Relayer{}).
		Complete(r)
}

func newPodForRelayer(cr *terrav1alpha1.Relayer) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}

	srcPort := "transfer"
	if cr.Spec.SrcPort != "" {
		srcPort = cr.Spec.SrcPort
	}

	dstPort := "transfer"
	if cr.Spec.DstPort != "" {
		dstPort = cr.Spec.DstPort
	}

	icsVersion := "ics20-1"
	if cr.Spec.ICSVersion != "" {
		icsVersion = cr.Spec.ICSVersion
	}

	firstMinGasAmount := "0"
	if cr.Spec.FirstNetwork.MinGasAmount != "" {
		firstMinGasAmount = cr.Spec.FirstNetwork.MinGasAmount
	}

	secondMinGasAmount := "0"
	if cr.Spec.SecondNetwork.MinGasAmount != "" {
		secondMinGasAmount = cr.Spec.SecondNetwork.MinGasAmount
	}

	envVars := []corev1.EnvVar{
		{
			Name:  "SRC_PORT",
			Value: srcPort,
		},
		{
			Name:  "DST_PORT",
			Value: dstPort,
		},
		{
			Name:  "VERSION",
			Value: icsVersion,
		},
		{
			Name:  "FIRST_NETWORK_NAME",
			Value: cr.Spec.FirstNetwork.NetworkName,
		},
		{
			Name:  "FIRST_COIN_TYPE",
			Value: cr.Spec.FirstNetwork.CoinType,
		},
		{
			Name:  "FIRST_GAS_ADJUSTMENT",
			Value: cr.Spec.FirstNetwork.GasAdjustment,
		},
		{
			Name:  "FIRST_GAS_PRICES",
			Value: cr.Spec.FirstNetwork.GasPrices,
		},
		{
			Name:  "FIRST_MIN_GAS_AMOUNT",
			Value: firstMinGasAmount,
		},
		{
			Name:  "FIRST_DEBUG",
			Value: strconv.FormatBool(cr.Spec.FirstNetwork.EnableDebug),
		},
		{
			Name:  "FIRST_MNEMONIC",
			Value: cr.Spec.FirstNetwork.RelayerKeyMnemonic,
		},
		{
			Name:  "SECOND_NETWORK_NAME",
			Value: cr.Spec.SecondNetwork.NetworkName,
		},
		{
			Name:  "SECOND_COIN_TYPE",
			Value: cr.Spec.SecondNetwork.CoinType,
		},
		{
			Name:  "SECOND_GAS_ADJUSTMENT",
			Value: cr.Spec.SecondNetwork.GasAdjustment,
		},
		{
			Name:  "SECOND_GAS_PRICES",
			Value: cr.Spec.SecondNetwork.GasPrices,
		},
		{
			Name:  "SECOND_MIN_GAS_AMOUNT",
			Value: secondMinGasAmount,
		},
		{
			Name:  "SECOND_DEBUG",
			Value: strconv.FormatBool(cr.Spec.SecondNetwork.EnableDebug),
		},
		{
			Name:  "SECOND_MNEMONIC",
			Value: cr.Spec.SecondNetwork.RelayerKeyMnemonic,
		},
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name,
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:            "relayer",
					Image:           cr.Spec.Container.Image,
					Env:             envVars,
					ImagePullPolicy: corev1.PullPolicy(cr.Spec.Container.ImagePullPolicy),
				},
			},
		},
	}

	return pod
}
