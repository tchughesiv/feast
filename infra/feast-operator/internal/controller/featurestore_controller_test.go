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

package controller

import (
	"context"
	"encoding/base64"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"gopkg.in/yaml.v3"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	feastdevv1alpha1 "github.com/feast-dev/feast/infra/feast-operator/api/v1alpha1"
	"github.com/feast-dev/feast/infra/feast-operator/internal/controller/services"
)

const feastProject = "test_project"

var _ = Describe("FeatureStore Controller", func() {
	Context("When reconciling a resource", func() {
		const resourceName = "test-resource"

		ctx := context.Background()

		typeNamespacedName := types.NamespacedName{
			Name:      resourceName,
			Namespace: "default",
		}
		featurestore := &feastdevv1alpha1.FeatureStore{}

		BeforeEach(func() {
			By("creating the custom resource for the Kind FeatureStore")
			err := k8sClient.Get(ctx, typeNamespacedName, featurestore)
			if err != nil && errors.IsNotFound(err) {
				resource := &feastdevv1alpha1.FeatureStore{
					ObjectMeta: metav1.ObjectMeta{
						Name:      resourceName,
						Namespace: "default",
					},
					Spec: feastdevv1alpha1.FeatureStoreSpec{FeastProject: feastProject},
				}
				Expect(k8sClient.Create(ctx, resource)).To(Succeed())
			}
		})

		AfterEach(func() {
			resource := &feastdevv1alpha1.FeatureStore{}
			err := k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			By("Cleanup the specific resource instance FeatureStore")
			Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
		})
		It("should successfully reconcile the resource", func() {
			By("Reconciling the created resource")
			controllerReconciler := &FeatureStoreReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &feastdevv1alpha1.FeatureStore{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			Expect(resource.Status).NotTo(BeNil())
			Expect(resource.Status.Applied.FeastProject).To(Equal(resource.Spec.FeastProject))
			Expect(resource.Status.Conditions).NotTo(BeEmpty())

			cond := resource.Status.Conditions[0]
			Expect(cond.Status).To(Equal(metav1.ConditionTrue))
			Expect(cond.Reason).To(Equal(feastdevv1alpha1.ReadyReason))
			Expect(cond.Type).To(Equal(feastdevv1alpha1.ReadyType))
			Expect(cond.Message).To(Equal(feastdevv1alpha1.ReadyMessage))
		})

		It("should properly encode a feature_store.yaml config", func() {
			By("Reconciling the created resource")
			controllerReconciler := &FeatureStoreReconciler{
				Client: k8sClient,
				Scheme: k8sClient.Scheme(),
			}

			_, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			Expect(err).NotTo(HaveOccurred())

			resource := &feastdevv1alpha1.FeatureStore{}
			err = k8sClient.Get(ctx, typeNamespacedName, resource)
			Expect(err).NotTo(HaveOccurred())

			feast := services.FeastServices{
				Client:       controllerReconciler.Client,
				Context:      ctx,
				Scheme:       controllerReconciler.Scheme,
				FeatureStore: resource,
			}
			deploy := &appsv1.Deployment{}
			err = k8sClient.Get(ctx, types.NamespacedName{Name: feast.GetFeastServiceName(services.RegistryFeastType), Namespace: resource.Namespace}, deploy)
			Expect(err).NotTo(HaveOccurred())

			Expect(deploy.Spec.Template.Spec.Containers).To(HaveLen(1))
			Expect(deploy.Spec.Template.Spec.Containers[0].Env).To(HaveLen(1))
			env := getEnvVar(services.FeatureStoreYamlEnvVar, deploy.Spec.Template.Spec.Containers[0].Env)
			Expect(env).NotTo(BeNil())

			fsYamlStr, err := feast.GetServiceFeatureStoreYamlBase64()
			Expect(err).NotTo(HaveOccurred())
			Expect(fsYamlStr).To(Equal(env.Value))

			envByte, err := base64.StdEncoding.DecodeString(env.Value)
			Expect(err).NotTo(HaveOccurred())
			repoConfig := &services.RepoConfig{}
			err = yaml.Unmarshal(envByte, repoConfig)
			Expect(err).NotTo(HaveOccurred())
			test := &services.RepoConfig{
				Project: feastProject,
				Registry: services.RegistryConfig{
					Path: "tmp/registry.db",
				},
				EntityKeySerializationVersion: feastdevv1alpha1.SerializationVersion,
			}
			Expect(repoConfig).To(Equal(test))
		})
	})
})

func getEnvVar(name string, envs []corev1.EnvVar) *corev1.EnvVar {
	for _, e := range envs {
		if e.Name == name {
			return &e
		}
	}
	return nil
}
