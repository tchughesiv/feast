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
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log"
)

func (feast *FeastServices) deployRegistry() error {
	if err := feast.createRegistryDeployment(); err != nil {
		return err
	}
	if err := feast.createRegistryService(); err != nil {
		return err
	}
	return nil
}

func (feast *FeastServices) createRegistryDeployment() error {
	logger := log.FromContext(feast.Context)
	deploy := &appsv1.Deployment{
		ObjectMeta: feast.GetObjectMeta(RegistryFeastType),
	}
	deploy.SetGroupVersionKind(appsv1.SchemeGroupVersion.WithKind("Deployment"))
	if op, err := controllerutil.CreateOrUpdate(feast.Context, feast.Client, deploy, controllerutil.MutateFn(func() error {
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
		ObjectMeta: feast.GetObjectMeta(RegistryFeastType),
	}
	svc.SetGroupVersionKind(corev1.SchemeGroupVersion.WithKind("Service"))
	if op, err := controllerutil.CreateOrUpdate(feast.Context, feast.Client, svc, controllerutil.MutateFn(func() error {
		return feast.setService(svc, RegistryFeastType)
	})); err != nil {
		return err
	} else if op == controllerutil.OperationResultCreated || op == controllerutil.OperationResultUpdated {
		logger.Info("Successfully reconciled", "Service", svc.Name, "operation", op)
	}
	return nil
}
