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
	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func GetRegistryObjects(cr *feastdevv1alpha1.FeatureStore) (objects []client.Object) {
	nsName := types.NamespacedName{Name: cr.Name, Namespace: cr.Namespace}
	objects = append(objects, getRegistryDeployment(nsName, cr.Status))
	objects = append(objects, getRegistryService(nsName, cr.Status))
	return objects
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

func getRegistryDeployment(nsName types.NamespacedName, status feastdevv1alpha1.FeatureStoreStatus) *appsv1.Deployment {
	deploy := &appsv1.Deployment{
		ObjectMeta: v1.ObjectMeta{Name: nsName.Name, Namespace: nsName.Namespace},
	}
	deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	return deploy
}

func getRegistryService(nsName types.NamespacedName, status feastdevv1alpha1.FeatureStoreStatus) *corev1.Service {
	service := &corev1.Service{
		ObjectMeta: v1.ObjectMeta{Name: nsName.Name, Namespace: nsName.Namespace},
	}
	service.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	return service
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
