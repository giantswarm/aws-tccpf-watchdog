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

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/giantswarm/microerror"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	capi "sigs.k8s.io/cluster-api/api/v1beta1"
	"sigs.k8s.io/cluster-api/util/annotations"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudformationservice "github.com/giantswarm/aws-tccpf-watchdog/pkg/cloud/services/cloudformation"
	"github.com/giantswarm/aws-tccpf-watchdog/pkg/key"
	"github.com/giantswarm/aws-tccpf-watchdog/pkg/utils"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	Client   client.Client
	CFClient cloudformation.CloudFormation
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=clusters.cluster.x-k8s.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("cluster", req.NamespacedName)

	cluster := &capi.Cluster{}
	err := r.Client.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, microerror.Mask(err)
	}

	// Return early if the object or Cluster is paused.
	if annotations.IsPaused(cluster, cluster) {
		log.Info("Cluster is marked as paused. Won't reconcile")
		return ctrl.Result{}, nil
	}

	// This controller does not need to clean up anything.
	if !cluster.DeletionTimestamp.IsZero() {
		log.Info("Cluster is deleted, skipping")
		return ctrl.Result{}, nil
	}

	// Check if this is a legacy Cluster CR.
	if !utils.IsLegacyCluster(log, *cluster) {
		log.Info("Cluster is not a legacy cluster, skipping")
		return ctrl.Result{}, nil
	}

	stackName := key.CFStackName(*cluster)
	log = log.WithValues("cfstack", stackName)

	service := cloudformationservice.NewService(log, r.CFClient)

	ok, err := service.CheckStackContainsAtLeastOneRouteDefinition(stackName)
	if IsAWSNotFound(err) {
		log.Info("CF stack not found")
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, microerror.Mask(err)
	}

	if ok {
		log.Info("Cloud formation stack contains at least one route definition")
		return ctrl.Result{}, nil
	}

	log.Info("Stack did not contain any route definition, deleting it")

	err = service.DeleteStack(stackName)
	if err != nil {
		return ctrl.Result{}, microerror.Mask(err)
	}

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&capi.Cluster{}).
		Complete(r)
}
