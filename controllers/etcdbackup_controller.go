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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	etcdv1alpha1 "github.com/SeasonPilot/etcd-operator/api/v1alpha1"
)

// EtcdBackupReconciler reconciles a EtcdBackup object
type EtcdBackupReconciler struct {
	client.Client
	Scheme    *runtime.Scheme
	BackImage string
}

// 设置实际任务 pod
func (r *EtcdBackupReconciler) setStateActual(ctx context.Context, state *backupState) error {
	var actual backupStateContainer

	key := client.ObjectKey{
		Namespace: state.backup.Namespace, // fixme: Namespace, Name 写反了
		Name:      state.backup.Name,
	}

	actual.pod = &corev1.Pod{}

	err := r.Get(ctx, key, actual.pod)
	if err != nil {
		if client.IgnoreNotFound(err) != nil {
			return err
		}
		actual.pod = nil
	}

	state.actual = &actual

	return nil
}

// 设置预期任务 pod
func (r *EtcdBackupReconciler) setStateDesire(state *backupState) error {
	var desired backupStateContainer

	pod := r.podForBackup(*state.backup, r.BackImage)

	err := ctrl.SetControllerReference(state.backup, pod, r.Scheme)
	if err != nil {
		return fmt.Errorf("setControllerReference err: %s", err)
	}

	desired.pod = pod
	state.desired = &desired
	return nil
}

func (r *EtcdBackupReconciler) podForBackup(backup etcdv1alpha1.EtcdBackup, image string) *corev1.Pod {
	var (
		endPoint  string
		secretRef *corev1.SecretEnvSource
		path      string
	)

	switch backup.Spec.StorageType {
	case etcdv1alpha1.EtcdBackupStorageTypeS3:
		endPoint = backup.Spec.S3.EndPoint
		secretRef = &corev1.SecretEnvSource{
			LocalObjectReference: corev1.LocalObjectReference{
				Name: backup.Spec.S3.Secret,
			},
		}
		path = backup.Spec.S3.Path
	case etcdv1alpha1.EtcdBackupStorageTypeOSS:
		endPoint = backup.Spec.OSS.EndPoint
		path = backup.Spec.OSS.Path
	}

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  backup.Name,
					Image: image,
					Args: []string{
						"--etcd-url", backup.Spec.EtcdURL, // fixme:
						"--backup-url", fmt.Sprintf("%s://%s", backup.Spec.StorageType, path), //fixme: 缺少 //
					},
					Env: []corev1.EnvVar{
						{
							Name:  "ENDPOINT",
							Value: endPoint,
						},
					},
					EnvFrom: []corev1.EnvFromSource{ // fixme:
						{
							SecretRef: secretRef,
						},
					},
					Resources: corev1.ResourceRequirements{
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"), // fixme: ResourceCPU
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("100Mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

// 获取 EtcdBackup CR 的状态
func (r *EtcdBackupReconciler) getState(ctx context.Context, req ctrl.Request) (*backupState, error) {
	var status = &backupState{} // 初始化一个空的 backupState
	status.backup = &etcdv1alpha1.EtcdBackup{}
	err := r.Get(ctx, req.NamespacedName, status.backup)
	if err != nil {
		return nil, client.IgnoreNotFound(err)
	}

	// fixme: 没有调用 r.setStateActual r.setStateDesire
	err = r.setStateActual(ctx, status)
	if err != nil {
		return nil, err
	}

	err = r.setStateDesire(status)
	if err != nil {
		return nil, err
	}

	return status, nil
}

type backupState struct {
	backup  *etcdv1alpha1.EtcdBackup
	actual  *backupStateContainer
	desired *backupStateContainer
}

type backupStateContainer struct {
	pod *corev1.Pod
}

//+kubebuilder:rbac:groups=etcd.season.io,resources=etcdbackups,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=etcd.season.io,resources=etcdbackups/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=etcd.season.io,resources=etcdbackups/finalizers,verbs=update

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the EtcdBackup object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.13.0/pkg/reconcile
func (r *EtcdBackupReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	l := log.FromContext(ctx)

	state, err := r.getState(ctx, req)
	if err != nil {
		return ctrl.Result{}, err
	}

	var action Action

	switch {
	case state.backup == nil: // CR 被删除
		l.Info("etcdBack Obj not found. Ignore...")
	case !state.backup.DeletionTimestamp.IsZero():
		l.Info("etcdBack Obj is deleted. Ignore...")
	case state.backup.Status.Phase == "":
		newBackup := state.backup.DeepCopy()
		newBackup.Status.Phase = etcdv1alpha1.EtcdBackupPhaseBackingup
		action = &PatchStatus{
			Client:   r.Client,
			original: state.backup,
			new:      newBackup,
		}
	case state.backup.Status.Phase == etcdv1alpha1.EtcdBackupPhaseFailed:
		l.Info("etcdBack has failed. Ignore...")
	case state.backup.Status.Phase == etcdv1alpha1.EtcdBackupPhaseCompleted:
		l.Info("Backup has completed. Ignoring.")

		// 执行 pod 相关
	case state.actual.pod == nil:
		action = CreateStatus{
			Client: r.Client,
			obj:    state.desired.pod, // fixme: 不是 &corev1.Pod{}
		}
	case state.actual.pod.Status.Phase == corev1.PodFailed:
		newBack := state.backup.DeepCopy()
		newBack.Status.Phase = etcdv1alpha1.EtcdBackupPhaseFailed
		action = PatchStatus{
			Client:   r.Client,
			original: state.backup,
			new:      newBack,
		}
	case state.actual.pod.Status.Phase == corev1.PodSucceeded:
		newBack := state.backup.DeepCopy()
		newBack.Status.Phase = etcdv1alpha1.EtcdBackupPhaseCompleted
		action = PatchStatus{
			Client:   r.Client,
			original: state.backup,
			new:      newBack,
		}
	}

	if action != nil {
		err = action.Execute(ctx)
		if err != nil {
			return ctrl.Result{}, err
		}
	}
	return ctrl.Result{}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *EtcdBackupReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&etcdv1alpha1.EtcdBackup{}).
		Owns(&corev1.Pod{}).
		Complete(r)
}
