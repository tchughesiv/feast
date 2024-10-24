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
	"strconv"
	"strings"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apimeta "k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

// Deploy the feast services
func (feast *FeastServices) Deploy() error {
	status := &feast.FeatureStore.Status
	appledSpec := status.Applied

	if appledSpec.Services != nil {
		if appledSpec.Services.OfflineStore != nil {
			if err := feast.deployFeastServiceType(OfflineFeastType); err != nil {
				return err
			}
		} else {
			apimeta.RemoveStatusCondition(&status.Conditions, feastdevv1alpha1.OfflineStoreReadyType)
			// if owned service objects exist, delete them ???
		}

		if appledSpec.Services.OnlineStore != nil {
			if err := feast.deployFeastServiceType(OnlineFeastType); err != nil {
				return err
			}
		} else {
			apimeta.RemoveStatusCondition(&status.Conditions, feastdevv1alpha1.OnlineStoreReadyType)
		}

		if appledSpec.Services.Registry != nil {
			if err := feast.deployFeastServiceType(RegistryFeastType); err != nil {
				return err
			}
		}
	}

	if err := feast.deployClient(); err != nil {
		return err
	}

	return nil
}

func (feast *FeastServices) deployFeastServiceType(feastType FeastServiceType) error {
	if err := feast.createDeployment(feastType); err != nil {
		return feast.setFeastServiceCondition(err, feastType)
	}
	if err := feast.createService(feastType); err != nil {
		return feast.setFeastServiceCondition(err, feastType)
	}

	return feast.setFeastServiceCondition(nil, feastType)
}

func (feast *FeastServices) createDeployment(feastType FeastServiceType) error {
	logger := log.FromContext(feast.Context)
	deploy := &appsv1.Deployment{
		ObjectMeta: feast.GetObjectMeta(feastType),
	}
	deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	if op, err := controllerutil.CreateOrUpdate(feast.Context, feast.Client, deploy, controllerutil.MutateFn(func() error {
		return feast.setDeployment(deploy, feastType)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "Deployment", deploy.Name, "operation", op)
	}

	return nil
}

func (feast *FeastServices) createService(feastType FeastServiceType) error {
	logger := log.FromContext(feast.Context)
	svc := &corev1.Service{
		ObjectMeta: feast.GetObjectMeta(feastType),
	}
	svc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	if op, err := controllerutil.CreateOrUpdate(feast.Context, feast.Client, svc, controllerutil.MutateFn(func() error {
		return feast.setService(svc, feastType)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "Service", svc.Name, "operation", op)
	}
	return nil
}

func (feast *FeastServices) setDeployment(deploy *appsv1.Deployment, feastType FeastServiceType) error {
	fsYamlB64, err := feast.GetServiceFeatureStoreYamlBase64(feastType)
	if err != nil {
		return err
	}
	deploy.Labels = feast.getLabels(feastType)
	deploySettings := FeastServiceConstants[feastType]
	serviceConfig := feast.getServiceConfig(feastType)

	// standard configs are applied here
	probeHandler := corev1.ProbeHandler{
		TCPSocket: &corev1.TCPSocketAction{
			Port: intstr.FromInt(int(deploySettings.TargetPort)),
		},
	}
	deploy.Spec = appsv1.DeploymentSpec{
		Replicas: &DefaultReplicas,
		Selector: metav1.SetAsLabelSelector(deploy.GetLabels()),
		Template: corev1.PodTemplateSpec{
			ObjectMeta: metav1.ObjectMeta{
				Labels: deploy.GetLabels(),
			},
			Spec: corev1.PodSpec{
				Containers: []corev1.Container{
					{
						Name:    string(feastType),
						Image:   *serviceConfig.Image,
						Command: deploySettings.Command,
						Ports: []corev1.ContainerPort{
							{
								Name:          string(feastType),
								ContainerPort: deploySettings.TargetPort,
								Protocol:      corev1.ProtocolTCP,
							},
						},
						Env: []corev1.EnvVar{
							{
								Name:  FeatureStoreYamlEnvVar,
								Value: fsYamlB64,
							},
						},
						LivenessProbe: &corev1.Probe{
							ProbeHandler:        probeHandler,
							InitialDelaySeconds: 30,
							PeriodSeconds:       30,
						},
						ReadinessProbe: &corev1.Probe{
							ProbeHandler:        probeHandler,
							InitialDelaySeconds: 20,
							PeriodSeconds:       10,
						},
					},
				},
			},
		},
	}

	// optional configs are applied here
	container := &deploy.Spec.Template.Spec.Containers[0]
	if serviceConfig.ImagePullPolicy != nil {
		container.ImagePullPolicy = *serviceConfig.ImagePullPolicy
	}
	if serviceConfig.Resources != nil {
		container.Resources = *serviceConfig.Resources
	}

	return controllerutil.SetControllerReference(feast.FeatureStore, deploy, feast.Scheme)
}

func (feast *FeastServices) setService(svc *corev1.Service, feastType FeastServiceType) error {
	svc.Labels = feast.getLabels(feastType)
	deploySettings := FeastServiceConstants[feastType]

	svc.Spec = corev1.ServiceSpec{
		Selector: svc.GetLabels(),
		Type:     corev1.ServiceTypeClusterIP,
		Ports: []corev1.ServicePort{
			{
				Name:       strings.ToLower(string(corev1.URISchemeHTTP)),
				Port:       HttpPort,
				Protocol:   corev1.ProtocolTCP,
				TargetPort: intstr.FromInt(int(deploySettings.TargetPort)),
			},
		},
	}

	hostname := svc.Name + "." + svc.Namespace + svcDomain
	hostnameWithPort := hostname + ":" + strconv.Itoa(HttpPort)
	if feastType == OfflineFeastType {
		feast.FeatureStore.Status.ServiceHostnames.OfflineStore = hostname
	}
	if feastType == OnlineFeastType {
		feast.FeatureStore.Status.ServiceHostnames.OnlineStore = strings.ToLower(string(corev1.URISchemeHTTP)) + "://" + hostnameWithPort
	}
	if feastType == RegistryFeastType {
		feast.FeatureStore.Status.ServiceHostnames.Registry = hostnameWithPort
	}

	return controllerutil.SetControllerReference(feast.FeatureStore, svc, feast.Scheme)
}

func (feast *FeastServices) getServiceConfig(feastType FeastServiceType) feastdevv1alpha1.ServiceConfig {
	appliedSpec := feast.FeatureStore.Status.Applied
	if feastType == OfflineFeastType && appliedSpec.Services.OfflineStore != nil {
		return appliedSpec.Services.OfflineStore.ServiceConfig
	}
	if feastType == OnlineFeastType && appliedSpec.Services.OnlineStore != nil {
		return appliedSpec.Services.OnlineStore.ServiceConfig
	}
	if feastType == RegistryFeastType && appliedSpec.Services.Registry != nil {
		return appliedSpec.Services.Registry.ServiceConfig
	}
	return feastdevv1alpha1.ServiceConfig{}
}

// GetObjectMeta returns the feast k8s object metadata
func (feast *FeastServices) GetObjectMeta(feastType FeastServiceType) metav1.ObjectMeta {
	return metav1.ObjectMeta{Name: feast.GetFeastServiceName(feastType), Namespace: feast.FeatureStore.Namespace}
}

// GetFeastServiceName returns the feast service object name based on service type
func (feast *FeastServices) GetFeastServiceName(feastType FeastServiceType) string {
	return feast.getFeastName() + "-" + string(feastType)
}

func (feast *FeastServices) getFeastName() string {
	return FeastPrefix + feast.FeatureStore.Name
}

func (feast *FeastServices) getLabels(feastType FeastServiceType) map[string]string {
	return map[string]string{
		feastdevv1alpha1.GroupVersion.Group + "/name":         feast.FeatureStore.Name,
		feastdevv1alpha1.GroupVersion.Group + "/service-type": string(feastType),
	}
}

func (feast *FeastServices) setFeastServiceCondition(err error, feastType FeastServiceType) error {
	logger := log.FromContext(feast.Context)
	conditionMap := FeastServiceConditions[feastType]
	if err != nil {
		cond := conditionMap[metav1.ConditionFalse]
		cond.Message = "Error: " + err.Error()
		apimeta.SetStatusCondition(&feast.FeatureStore.Status.Conditions, cond)
		logger.Error(err, "Error deploying the FeatureStore "+string(ClientFeastType)+" service")
		return err
	} else {
		apimeta.SetStatusCondition(&feast.FeatureStore.Status.Conditions, conditionMap[metav1.ConditionTrue])
	}
	return nil
}
