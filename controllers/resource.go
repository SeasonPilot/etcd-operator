package controllers

import (
	etcdv1alpha1 "github.com/SeasonPilot/etcd-operator/api/v1alpha1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func mutateSts(cluster *etcdv1alpha1.EtcdCluster, sts *appsv1.StatefulSet) {

}

func mutateSvc(cluster *etcdv1alpha1.EtcdCluster, svc *corev1.Service) {

}
