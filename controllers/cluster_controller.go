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

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/credentials/stscreds"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudformation"
	"github.com/go-errors/errors"
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
	ARN    string
	Client client.Client
	Log    logr.Logger
	Region string
	Scheme *runtime.Scheme
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
		return reconcile.Result{}, err
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

	cloudFormation, err := r.getCFClient(r.Region, r.ARN)
	if err != nil {
		return reconcile.Result{}, err
	}

	stackName := key.CFStackName(*cluster)
	log = log.WithValues("cfstack", stackName)

	service := cloudformationservice.NewService(log, *cloudFormation)

	ok, err := service.CheckStackHasAllResources(stackName)
	if IsAWSNotFound(err) {
		log.Info("CF stack not found")
		return ctrl.Result{}, nil
	} else if err != nil {
		return ctrl.Result{}, errors.Wrap(err, 1)
	}

	if ok {
		log.Info("Cloud formation stack resources were all found")
		return ctrl.Result{}, nil
	}

	log.Info("Stack did not have all required resources, deleting it")

	err = service.DeleteStack(stackName)
	if err != nil {
		return ctrl.Result{}, err
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

func (r *ClusterReconciler) getCFClient(region, arn string) (*cloudformation.CloudFormation, error) {
	ns, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	if err != nil {
		return nil, err
	}

	cnf := &aws.Config{}
	if arn != "" {
		cnf.Credentials = stscreds.NewCredentials(ns, arn)
	}
	cfClient := cloudformation.New(ns, cnf)

	return cfClient, nil
}
