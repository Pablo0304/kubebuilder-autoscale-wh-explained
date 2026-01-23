/*
Copyright 2026.

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

package controller

import (
	"context"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	staticv1 "staticwebsite/api/v1"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// StaticSiteReconciler reconciles a StaticSite object
type StaticSiteReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=static.static.com,resources=staticsites,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=static.static.com,resources=staticsites/status,verbs=get;update;patch
// +kubebuilder:rbac:groups=static.static.com,resources=staticsites/finalizers,verbs=update
// +kubebuilder:rbac:groups=apps,resources=deployments,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=services,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups="",resources=configmaps,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=networking.k8s.io,resources=ingresses,verbs=get;list;watch;create;update;patch;delete

// Reconcile is part of the main kubernetes reconciliation loop which aims to
// move the current state of the cluster closer to the desired state.
// TODO(user): Modify the Reconcile function to compare the state specified by
// the StaticSite object against the actual cluster state, and then
// perform operations to make the cluster state reflect the state specified by
// the user.
//
// For more details, check Reconcile and its Result here:
// - https://pkg.go.dev/sigs.k8s.io/controller-runtime@v0.22.4/pkg/reconcile
func (r *StaticSiteReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {
	var site staticv1.StaticSite
	if err := r.Get(ctx, req.NamespacedName, &site); err != nil {
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	depName := site.Name + "-web"
	svcName := site.Name + "-svc"
	cfgName := site.Name + "-nginx-config"
	labels := map[string]string{
		"app":        "staticsite",
		"staticsite": site.Name,
	}

	desiredConfig := buildNginxConfigMap(&site, cfgName, labels)
	if err := controllerutil.SetControllerReference(&site, desiredConfig, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := createOrUpdateConfigMap(ctx, r.Client, desiredConfig); err != nil {
		return ctrl.Result{}, err
	}

	desiredDeployment := buildDeployment(&site, depName, labels)
	if err := controllerutil.SetControllerReference(&site, desiredDeployment, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := createOrUpdateDeployment(ctx, r.Client, desiredDeployment); err != nil {
		return ctrl.Result{}, err
	}

	desiredService := buildService(&site, svcName, labels)
	if err := controllerutil.SetControllerReference(&site, desiredService, r.Scheme); err != nil {
		return ctrl.Result{}, err
	}
	if err := createOrUpdateService(ctx, r.Client, desiredService); err != nil {
		return ctrl.Result{}, err
	}

	return ctrl.Result{}, nil
}

func buildNginxConfigMap(site *staticv1.StaticSite, name string, labels map[string]string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: site.Namespace,
			Labels:    labels,
		},
		Data: map[string]string{
			"default.conf": `server {
  listen 80;
  location / {
    root   /usr/share/nginx/html;
    index  index.html index.htm;
  }
}

server {
  listen 8081;
  location /stub_status {
    stub_status;
    allow 127.0.0.1;
    deny all;
  }
}
`,
		},
	}
}

func buildDeployment(site *staticv1.StaticSite, name string, labels map[string]string) *appsv1.Deployment {
	containerPort := int32(80)
	if site.Spec.Port != 0 {
		containerPort = site.Spec.Port
	}
	metricsPort := int32(9113)

	return &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: site.Namespace,
			Labels:    labels,
		},
		Spec: appsv1.DeploymentSpec{
			Selector: &metav1.LabelSelector{MatchLabels: labels},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{Labels: labels},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  "web",
							Image: site.Spec.Image,
							Ports: []corev1.ContainerPort{{ContainerPort: containerPort}},
							VolumeMounts: []corev1.VolumeMount{
								{
									Name:      "nginx-config",
									MountPath: "/etc/nginx/conf.d/default.conf",
									SubPath:   "default.conf",
								},
							},
						},
						{
							Name:  "metrics",
							Image: "nginx/nginx-prometheus-exporter:latest",
							Args:  []string{"-nginx.scrape-uri=http://127.0.0.1:8081/stub_status"},
							Ports: []corev1.ContainerPort{{ContainerPort: metricsPort}},
						},
					},
					Volumes: []corev1.Volume{
						{
							Name: "nginx-config",
							VolumeSource: corev1.VolumeSource{
								ConfigMap: &corev1.ConfigMapVolumeSource{
									LocalObjectReference: corev1.LocalObjectReference{
										Name: site.Name + "-nginx-config",
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func buildService(site *staticv1.StaticSite, name string, labels map[string]string) *corev1.Service {
	servicePort := int32(80)
	if site.Spec.Service.Port != 0 {
		servicePort = site.Spec.Service.Port
	}
	metricsPort := int32(9113)
	var httpNodePort int32
	var metricsNodePort int32
	if site.Spec.Service.Type == corev1.ServiceTypeNodePort {
		httpNodePort = 30080
		metricsNodePort = 30913
	}
	return &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: site.Namespace,
			Labels:    labels,
		},
		Spec: corev1.ServiceSpec{
			Type:     site.Spec.Service.Type,
			Selector: labels,
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       servicePort,
					TargetPort: intstr.FromInt32(servicePort),
					NodePort:   httpNodePort,
				},
				{
					Name:       "metrics",
					Port:       metricsPort,
					TargetPort: intstr.FromInt32(metricsPort),
					NodePort:   metricsNodePort,
				},
			},
		},
	}
}

func createOrUpdateDeployment(ctx context.Context, c client.Client, desired *appsv1.Deployment) error {
	existing := &appsv1.Deployment{}
	err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, desired)
		}
		return err
	}
	existing.Labels = desired.Labels
	existing.Spec = desired.Spec
	return c.Update(ctx, existing)
}

func createOrUpdateService(ctx context.Context, c client.Client, desired *corev1.Service) error {
	existing := &corev1.Service{}
	err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, desired)
		}
		return err
	}
	existing.Labels = desired.Labels
	existing.Spec.Type = desired.Spec.Type
	existing.Spec.Selector = desired.Spec.Selector
	existing.Spec.Ports = desired.Spec.Ports
	return c.Update(ctx, existing)
}

func createOrUpdateConfigMap(ctx context.Context, c client.Client, desired *corev1.ConfigMap) error {
	existing := &corev1.ConfigMap{}
	err := c.Get(ctx, client.ObjectKeyFromObject(desired), existing)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return c.Create(ctx, desired)
		}
		return err
	}
	existing.Labels = desired.Labels
	existing.Data = desired.Data
	return c.Update(ctx, existing)
}

// SetupWithManager sets up the controller with the Manager.
func (r *StaticSiteReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&staticv1.StaticSite{}).
		Named("staticsite").
		Complete(r)
}
