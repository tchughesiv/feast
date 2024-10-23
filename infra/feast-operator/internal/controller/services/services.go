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
	"strconv"
	"strings"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	"gopkg.in/yaml.v3"
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
	logger := log.FromContext(feast.Context)
	status := &feast.FeatureStore.Status
	appledSpec := status.Applied

	if appledSpec.Services != nil {
		if appledSpec.Services.Registry != nil {
			if err := feast.deployRegistry(); err != nil {
				apimeta.SetStatusCondition(&status.Conditions, metav1.Condition{
					Type:    feastdevv1alpha1.RegistryReadyType,
					Status:  metav1.ConditionFalse,
					Reason:  feastdevv1alpha1.RegistryFailedReason,
					Message: "Error: " + err.Error(),
				})
				logger.Error(err, "Error deploying the FeatureStore "+string(RegistryFeastType)+" service")
				return err
			} else {
				apimeta.SetStatusCondition(&status.Conditions, metav1.Condition{
					Type:    feastdevv1alpha1.RegistryReadyType,
					Status:  metav1.ConditionTrue,
					Reason:  feastdevv1alpha1.ReadyReason,
					Message: feastdevv1alpha1.RegistryReadyMessage,
				})
			}
		} // else {
		// if apimeta.RemoveStatusCondition(&status.Conditions, feastdevv1alpha1.RegistryReadyType) {
		// if owned service objects exist, delete them ???
		// }

		// since registry service is required, not needed for this service? but for the others def. needed
		// }
	}

	if err := feast.deployClient(); err != nil {
		apimeta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    feastdevv1alpha1.ClientReadyType,
			Status:  metav1.ConditionFalse,
			Reason:  feastdevv1alpha1.ClientFailedReason,
			Message: "Error: " + err.Error(),
		})
		logger.Error(err, "Error deploying the FeatureStore "+string(ClientFeastType)+" service")
		return err
	} else {
		apimeta.SetStatusCondition(&status.Conditions, metav1.Condition{
			Type:    feastdevv1alpha1.ClientReadyType,
			Status:  metav1.ConditionTrue,
			Reason:  feastdevv1alpha1.ReadyReason,
			Message: feastdevv1alpha1.ClientReadyMessage,
		})
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

	// required configs are applied here
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
		feast.FeatureStore.Status.ServiceUrls.OfflineStore = hostname
	}
	if feastType == OnlineFeastType {
		feast.FeatureStore.Status.ServiceUrls.OnlineStore = strings.ToLower(string(corev1.URISchemeHTTP)) + "://" + hostnameWithPort
	}
	if feastType == RegistryFeastType {
		feast.FeatureStore.Status.ServiceUrls.Registry = hostnameWithPort
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

func (feast *FeastServices) getLabels(feastType FeastServiceType) map[string]string {
	return map[string]string{
		feastdevv1alpha1.GroupVersion.Group + "/name":         feast.FeatureStore.Name,
		feastdevv1alpha1.GroupVersion.Group + "/service-type": string(feastType),
	}
}

func (feast *FeastServices) getFeastName() string {
	return FeastPrefix + feast.FeatureStore.Name
}

// GetFeastServiceName returns the feast service object name based on service type
func (feast *FeastServices) GetFeastServiceName(feastType FeastServiceType) string {
	return feast.getFeastName() + "-" + string(feastType)
}

// GetServiceFeatureStoreYamlBase64 returns a base64 encoded feature_store.yaml config for the feast service
func (feast *FeastServices) GetServiceFeatureStoreYamlBase64(feastType FeastServiceType) (string, error) {
	fsYaml, err := feast.getServiceFeatureStoreYaml(feastType)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(fsYaml), nil
}

func (feast *FeastServices) getServiceFeatureStoreYaml(feastType FeastServiceType) ([]byte, error) {
	return yaml.Marshal(feast.getServiceRepoConfig(feastType))
}

func (feast *FeastServices) getServiceRepoConfig(feastType FeastServiceType) RepoConfig {
	appliedSpec := feast.FeatureStore.Status.Applied
	repoConfig := feast.getClientRepoConfig()
	if appliedSpec.Services != nil {
		if appliedSpec.Services.OfflineStore != nil && feastType == OfflineFeastType {
			repoConfig.OfflineStore = OfflineStoreConfig{
				Type: OfflineDaskConfigType,
				// ?? Path: LocalRegistryPath,
			}
		}
		if appliedSpec.Services.OnlineStore != nil {
			repoConfig.OnlineStore = OnlineStoreConfig{
				Type: OnlineSqliteConfigType,
				Path: LocalOnlinePath,
			}
		}
		if appliedSpec.Services.Registry != nil {
			repoConfig.Registry = RegistryConfig{
				RegistryType: RegistryFileConfigType,
				Path:         LocalRegistryPath,
			}
		}
	}
	return repoConfig
}
