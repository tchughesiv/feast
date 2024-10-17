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
	"encoding/base64"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	controllerruntime "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// DeployRegistry
func (feast *FeastServices) DeployRegistry() error {
	if err := feast.createRegistryDeployment(); err != nil {
		return err
	}
	if err := feast.createRegistryService(); err != nil {
		return err
	}
	if err := feast.createClientConfigMap(); err != nil {
		return err
	}
	return nil
}

func (feast *FeastServices) createRegistryDeployment() error {
	logger := log.FromContext(feast.Context)
	deploy := &appsv1.Deployment{
		ObjectMeta: feast.getObjectMeta(RegistryFeastType),
	}
	deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	if err := controllerruntime.SetControllerReference(feast.FeatureStore, deploy, feast.Scheme); err != nil {
		return err
	}
	if op, err := controllerruntime.CreateOrUpdate(feast.Context, feast.Client, deploy, controllerutil.MutateFn(func() error {
		return feast.setDeployment(deploy, RegistryFeastType)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "Deployment", deploy.Name, "operation", op)
	}
	return nil
}

func (feast *FeastServices) createRegistryService() error {
	logger := log.FromContext(feast.Context)
	svc := &corev1.Service{
		ObjectMeta: feast.getObjectMeta(RegistryFeastType),
	}
	svc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	if err := controllerruntime.SetControllerReference(feast.FeatureStore, svc, feast.Scheme); err != nil {
		return err
	}
	if op, err := controllerruntime.CreateOrUpdate(feast.Context, feast.Client, svc, controllerutil.MutateFn(func() error {
		feast.setService(svc, RegistryFeastType)
		return nil
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "Service", svc.Name, "operation", op)
	}
	return nil
}

func (feast *FeastServices) createClientConfigMap() error {
	logger := log.FromContext(feast.Context)
	cm := &corev1.ConfigMap{
		ObjectMeta: feast.getObjectMeta(ClientFeastType),
	}
	cm.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("ConfigMap"))
	if err := controllerruntime.SetControllerReference(feast.FeatureStore, cm, feast.Scheme); err != nil {
		return err
	}
	if op, err := controllerruntime.CreateOrUpdate(feast.Context, feast.Client, cm, controllerutil.MutateFn(func() error {
		return feast.setClientConfigMap(cm)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "ConfigMap", cm.Name, "operation", op)
	}
	return nil
}

func (feast *FeastServices) setDeployment(deploy *appsv1.Deployment, feastType FeastServiceType) error {
	fsYamlB64, err := feast.GetServiceFeatureStoreYamlBase64()
	if err != nil {
		return err
	}
	replicas := int32(1)
	deploy.Labels = feast.getLabels(feastType)
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: &replicas,
		Selector: v1.SetAsLabelSelector(deploy.GetLabels()),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: v1.ObjectMeta{
				Labels: deploy.GetLabels(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:            string(feastType) + "-server",
						Image:           "feastdev/feature-server:" + feast.FeatureStore.Status.FeastVersion,
						ImagePullPolicy: corev1.PullIfNotPresent,
						Env: []corev1.EnvVar{
							{
								Name:  FeatureStoreYamlEnvVar,
								Value: fsYamlB64,
							},
						},
					},
				},
			},
		},
	}
	if feastType == RegistryFeastType {
		deploy.Spec.Template.Spec.Containers[0].Command = []string{"feast", "serve_registry"}
		deploy.Spec.Template.Spec.Containers[0].Ports = []corev1.ContainerPort{
			{
				Name:          string(feastType),
				ContainerPort: RegistryPort,
				Protocol:      corev1.ProtocolTCP,
			},
		}
		probeHandler := corev1.ProbeHandler{
			TCPSocket: &corev1.TCPSocketAction{
				Port: intstr.FromInt(int(RegistryPort)),
			},
		}
		deploy.Spec.Template.Spec.Containers[0].LivenessProbe = &corev1.Probe{
			ProbeHandler:        probeHandler,
			InitialDelaySeconds: 30,
			PeriodSeconds:       30,
		}
		deploy.Spec.Template.Spec.Containers[0].ReadinessProbe = &corev1.Probe{
			ProbeHandler:        probeHandler,
			InitialDelaySeconds: 20,
			PeriodSeconds:       10,
		}
	}
	return nil
}

func (feast *FeastServices) setService(svc *corev1.Service, feastType FeastServiceType) {
	svc.Labels = feast.getLabels(feastType)
	svc.Spec = corev1.ServiceSpec{
		Selector: svc.GetLabels(),
		Type:     corev1.ServiceTypeClusterIP,
	}
	if feastType == RegistryFeastType {
		svc.Spec.Ports = []corev1.ServicePort{
			{
				Name:       "http",
				Port:       int32(80),
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(RegistryPort)),
			},
		}
		feast.FeatureStore.Status.ServiceUrls.Registry = svc.Name + "." + svc.Namespace + ".svc.cluster.local:80"
	}
}

func (feast *FeastServices) setClientConfigMap(cm *corev1.ConfigMap) error {
	cm.Labels = feast.getLabels(ClientFeastType)
	clientYaml, err := feast.getClientFeatureStoreYaml()
	if err != nil {
		return err
	}
	cm.BinaryData = map[string][]byte{"feature_store.yaml": clientYaml}
	feast.FeatureStore.Status.ClientConfigMap = cm.Name
	return nil
}

func (feast *FeastServices) getObjectMeta(feastType FeastServiceType) v1.ObjectMeta {
	return v1.ObjectMeta{Name: feast.GetFeastServiceName(feastType), Namespace: feast.FeatureStore.Namespace}
}

func (feast *FeastServices) getLabels(feastType FeastServiceType) map[string]string {
	return map[string]string{
		feastdevv1alpha1.GroupVersion.Group + "/name":    feast.getFeastName(),
		feastdevv1alpha1.GroupVersion.Group + "/service": feast.GetFeastServiceName(feastType),
	}
}

func (feast *FeastServices) getFeastName() string {
	return FeastPrefix + feast.FeatureStore.Name
}

func (feast *FeastServices) GetFeastServiceName(feastType FeastServiceType) string {
	return feast.getFeastName() + "-" + string(feastType)
}

func (feast *FeastServices) GetServiceFeatureStoreYamlBase64() (string, error) {
	fsYaml, err := feast.getServiceFeatureStoreYaml()
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(fsYaml), nil
}

func (feast *FeastServices) getServiceFeatureStoreYaml() ([]byte, error) {
	return yaml.Marshal(feast.getServiceRepoConfig())
}

func (feast *FeastServices) getServiceRepoConfig() RepoConfig {
	appliedSpec := feast.FeatureStore.Status.Applied
	return RepoConfig{
		Project:  appliedSpec.FeastProject,
		Provider: LocalProviderType,
		Registry: RegistryConfig{
			RegistryType: RegistryFileConfigType,
			Path:         "tmp/registry.db",
		},
		EntityKeySerializationVersion: feastdevv1alpha1.SerializationVersion,
	}
}

func (feast *FeastServices) getClientFeatureStoreYaml() ([]byte, error) {
	return yaml.Marshal(feast.getClientRepoConfig())
}

func (feast *FeastServices) getClientRepoConfig() RepoConfig {
	status := feast.FeatureStore.Status
	clientRepoConfig := RepoConfig{
		Project:                       status.Applied.FeastProject,
		EntityKeySerializationVersion: feastdevv1alpha1.SerializationVersion,
	}
	if len(status.ServiceUrls.Registry) > 0 {
		clientRepoConfig.Registry = RegistryConfig{
			RegistryType: RegistryRemoteConfigType,
			Path:         status.ServiceUrls.Registry,
		}
	}
	return clientRepoConfig
}
