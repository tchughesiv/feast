/*
Copyright 2024 Feast Community.

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

package services

import (
	"context"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
)

const FeastPrefix = "feast-"
const RegistryTypeSuffix FeastTypeSuffix = "-registry"

type FeastTypeSuffix string

type FeastServices struct {
	client.Client
	Context      context.Context
	Scheme       *runtime.Scheme
	FeatureStore *feastdevv1alpha1.FeatureStore
}

func (feast *FeastServices) DeployRegistry() error {
	if _, err := feast.createRegistryDeployment(); err != nil {
		return err
	}
	if _, err := feast.createRegistryService(); err != nil {
		return err
	}
	return nil
}

/*
apiVersion: apps/v1
kind: Deployment
metadata:
  name: feast-registry-server-feast-feature-server
  labels:
    app.kubernetes.io/name: feast-feature-server
    app.kubernetes.io/instance: feast-registry-server
spec:
  replicas: 1
  selector:
    matchLabels:
      app.kubernetes.io/name: feast-feature-server
      app.kubernetes.io/instance: feast-registry-server
  template:
    metadata:
      labels:
        app.kubernetes.io/name: feast-feature-server
        app.kubernetes.io/instance: feast-registry-server
    spec:
      containers:
        - name: feast-feature-server
          image: "feastdev/feature-server:0.40.1"
          imagePullPolicy: IfNotPresent
          env:
            - name: FEATURE_STORE_YAML_BASE64
              value: "??"
          command:
            - "feast"
            - "serve_registry"
          ports:
            - name: registry
              containerPort: 6570
              protocol: TCP
          livenessProbe:
            tcpSocket:
              port: registry
            initialDelaySeconds: 30
            periodSeconds: 30
          readinessProbe:
            tcpSocket:
              port: registry
            initialDelaySeconds: 20
            periodSeconds: 10
*/

func (feast *FeastServices) createRegistryDeployment() (controllerutil.OperationResult, error) {
	// appliedSpec := feast.FeatureStore.Status.Applied
	name := feast.getName(RegistryTypeSuffix)
	deploy := &appsv1.Deployment{
		ObjectMeta: feast.getObjectMeta(),
	}
	deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	return controllerruntime.CreateOrUpdate(feast.Context, feast.Client, deploy, controllerutil.MutateFn(func() error {
		replicas := int32(1)
		deploy.Labels = getLabels(name)
		deploy.Spec = appsv1.DeploymentSpec{
			Replicas: &replicas,
			Selector: v1.SetAsLabelSelector(getLabels(name)),
			Template: corev1.PodTemplateSpec{
				ObjectMeta: v1.ObjectMeta{
					Labels: getLabels(name),
				},
				Spec: corev1.PodSpec{
					Containers: []corev1.Container{
						{
							Name:  name,
							Image: "feastdev/feature-server:" + feastdevv1alpha1.Version,
						},
					},
				},
			},
		}
		return nil
	}))
}

func (feast *FeastServices) createRegistryService() (controllerutil.OperationResult, error) {
	name := feast.getName(RegistryTypeSuffix)
	service := &corev1.Service{
		ObjectMeta: feast.getObjectMeta(),
	}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	return controllerruntime.CreateOrUpdate(feast.Context, feast.Client, service, controllerutil.MutateFn(func() error {
		service.Labels = getLabels(name)
		service.Spec = corev1.ServiceSpec{}
		return nil
	}))
}

/*
apiVersion: v1
kind: Service
metadata:
  name: feast-registry-server-feast-feature-server
  labels:
    app.kubernetes.io/name: feast-feature-server
    app.kubernetes.io/instance: feast-registry-server
spec:
  type: ClusterIP
  ports:
    - port: 80
      targetPort: registry
      protocol: TCP
      name: http
  selector:
    app.kubernetes.io/name: feast-feature-server
    app.kubernetes.io/instance: feast-registry-server
*/

func (feast *FeastServices) getObjectMeta() v1.ObjectMeta {
	return v1.ObjectMeta{Name: feast.FeatureStore.Name, Namespace: feast.FeatureStore.Namespace}
}

func getLabels(name string) map[string]string {
	return map[string]string{
		feastdevv1alpha1.GroupVersion.Group + "/name": name,
	}
}

func (feast *FeastServices) getName(typeSuffix FeastTypeSuffix) string {
	return FeastPrefix + feast.FeatureStore.Name + string(typeSuffix)
}
