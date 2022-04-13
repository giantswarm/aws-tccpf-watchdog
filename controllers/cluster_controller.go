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
	"time"

	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/giantswarm/apiextensions/v6/pkg/apis/infrastructure/v1alpha3"
	"github.com/giantswarm/microerror"
	"github.com/go-logr/logr"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	cloudformationservice "github.com/giantswarm/aws-tccpf-watchdog/pkg/cloud/services/cloudformation"
	"github.com/giantswarm/aws-tccpf-watchdog/pkg/key"
)

// ClusterReconciler reconciles a Cluster object
type ClusterReconciler struct {
	CFClient cloudformation.CloudFormation
	Client   client.Client
	Log      logr.Logger
	Scheme   *runtime.Scheme
}

//+kubebuilder:rbac:groups=clusters.cluster.x-k8s.io,resources=clusters,verbs=get;list;watch;create;update;patch;delete

func (r *ClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	log := r.Log.WithValues("awsCluster", req.NamespacedName)

	awsCluster := &v1alpha3.AWSCluster{}
	err := r.Client.Get(ctx, req.NamespacedName, awsCluster)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return reconcile.Result{}, nil
		}
		return reconcile.Result{}, microerror.Mask(err)
	}

	// This controller does not need to clean up anything.
	if !awsCluster.DeletionTimestamp.IsZero() {
		log.Info("Cluster is deleted, skipping")
		return ctrl.Result{}, nil
	}

	stackName := key.CFStackName(*awsCluster)
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

	return ctrl.Result{
		RequeueAfter: 10 * time.Minute,
	}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *ClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		// Uncomment the following line adding a pointer to an instance of the controlled resource as an argument
		For(&v1alpha3.AWSCluster{}).
		Complete(r)
}
