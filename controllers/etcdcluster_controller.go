/*
Copyright 2022 season.

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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1alpha1 "github.com/SeasonPilot/etcd-operator/api/v1alpha1"
)

// EtcdClusterReconciler reconciles a EtcdCluster object
type EtcdClusterReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// fixme: 小写复数

//+kubebuilder:rbac:groups=etcd.season.io,resources=etcdclusters,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=apps,resources=statefulsets,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=core,resources=services,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.season.io,resources=etcdclusters/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.season.io,resources=etcdclusters/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdCluster object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *EtcdClusterReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	//var cluster *etcdv1alpha1.EtcdCluster // fixme: nil
	var cluster = &etcdv1alpha1.EtcdCluster{}
	err := r.Get(ctx, req.NamespacedName, cluster)
	if err != nil {
		l.Error(err, "Get EtcdCluster failed", "EtcdCluster", cluster)
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	l.Info("Get EtcdCluster success")

	var headlessSvc = &corev1.Service{}
	headlessSvc.Namespace = cluster.Namespace
	headlessSvc.Name = cluster.Name
	or, err := ctrl.CreateOrUpdate(ctx, r.Client, headlessSvc, func() error {
		mutateSvc(cluster, headlessSvc)
		return controllerutil.SetControllerReference(cluster, headlessSvc, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("CreateOrUpdate Headless Service", "Result", or)

	var sts = &appsv1.StatefulSet{}
	sts.Namespace = cluster.Namespace
	sts.Name = cluster.Name
	or, err = ctrl.CreateOrUpdate(ctx, r.Client, sts, func() error {
		mutateSts(cluster, sts)
		return controllerutil.SetControllerReference(cluster, sts, r.Scheme)
	})
	if err != nil {
		return ctrl.Result{}, err
	}
	l.Info("CreateOrUpdate StatefulSet", "Result", or)

	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdClusterReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdCluster{}).
		Owns(&appsv1.StatefulSet{}).
		Owns(&corev1.Service{}).
		Complete(r)
}
