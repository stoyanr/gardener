// Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package extensions

import (
	"time"

	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsinstall "github.com/gardener/gardener/pkg/apis/extensions/install"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
)

var (
	scheme *runtime.Scheme
)

func init() {
	scheme = runtime.NewScheme()
	extensionsinstall.Install(scheme)
}

func mkUnstructuredAccessor(obj extensionsv1alpha1.Object) extensionsv1alpha1.Object {
	u := &unstructured.Unstructured{}
	Expect(scheme.Convert(obj, u, nil)).To(Succeed())
	return UnstructuredAccessor(u)
}

func mkUnstructuredAccessorWithSpec(spec extensionsv1alpha1.DefaultSpec) extensionsv1alpha1.Spec {
	return mkUnstructuredAccessor(&extensionsv1alpha1.Infrastructure{Spec: extensionsv1alpha1.InfrastructureSpec{DefaultSpec: spec}}).GetExtensionSpec()
}

func mkUnstructuredAccessorWithStatus(status extensionsv1alpha1.DefaultStatus) extensionsv1alpha1.Status {
	return mkUnstructuredAccessor(&extensionsv1alpha1.Infrastructure{Status: extensionsv1alpha1.InfrastructureStatus{DefaultStatus: status}}).GetExtensionStatus()
}

var _ = Describe("Accessor", func() {
	Describe("#Accessor", func() {
		It("should create an accessor for extensions", func() {
			extension := &extensionsv1alpha1.Infrastructure{}
			acc, err := Accessor(extension)

			Expect(err).NotTo(HaveOccurred())
			Expect(acc).To(BeIdenticalTo(extension))
		})

		It("should create an unstructured accessor for unstructured", func() {
			u := &unstructured.Unstructured{}
			acc, err := Accessor(u)

			Expect(err).NotTo(HaveOccurred())
			Expect(acc).To(Equal(UnstructuredAccessor(u)))
		})

		It("should error for other objects", func() {
			_, err := Accessor(&corev1.ConfigMap{})

			Expect(err).To(HaveOccurred())
		})
	})

	Context("#UnstructuredAccessor", func() {
		Context("#GetExtensionSpec", func() {
			Describe("#GetExtensionType", func() {
				It("should get the extension type", func() {
					var (
						t   = "foo"
						acc = mkUnstructuredAccessorWithSpec(extensionsv1alpha1.DefaultSpec{Type: t})
					)

					Expect(acc.GetExtensionType()).To(Equal(t))
				})
			})

			Describe("#GetProviderConfig", func() {
				It("should get the provider config", func() {
					var (
						pc = &runtime.RawExtension{
							Object: &corev1.Secret{},
						}
						acc = mkUnstructuredAccessorWithSpec(extensionsv1alpha1.DefaultSpec{ProviderConfig: pc})
					)

					Expect(acc.GetProviderConfig()).To(Equal(&runtime.RawExtension{
						Raw: []byte(`{"metadata":{"creationTimestamp":null}}`),
					}))
				})

				It("should return nil", func() {
					acc := mkUnstructuredAccessorWithSpec(extensionsv1alpha1.DefaultSpec{})

					Expect(acc.GetProviderConfig()).To(BeNil())
				})
			})

			Describe("#GetResources", func() {
				It("should get the resources", func() {
					var (
						resources = []gardencorev1alpha1.NamedResourceReference{
							{
								Name: "test",
								ResourceRef: autoscalingv1.CrossVersionObjectReference{
									Kind:       "Secret",
									Name:       "test-secret",
									APIVersion: "v1",
								},
							},
						}
						acc = mkUnstructuredAccessorWithSpec(extensionsv1alpha1.DefaultSpec{Resources: resources})
					)
					getResources := acc.GetResources()
					Expect(getResources).To(Equal(resources))
				})
			})
		})

		Context("#GetExtensionStatus", func() {
			Describe("#GetProviderStatus", func() {
				It("should get the provider status", func() {
					var (
						ps = &runtime.RawExtension{
							Object: &corev1.Secret{},
						}
						acc = mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{ProviderStatus: ps})
					)

					Expect(acc.GetProviderStatus()).To(Equal(&runtime.RawExtension{
						Raw: []byte(`{"metadata":{"creationTimestamp":null}}`),
					}))
				})

				It("should return nil", func() {
					acc := mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{})

					Expect(acc.GetProviderStatus()).To(BeNil())
				})
			})

			Describe("#GetLastOperation", func() {
				It("should get the last operation", func() {
					var (
						desc = "desc"
						acc  = mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{LastOperation: &gardencorev1beta1.LastOperation{Description: "desc"}})
					)

					Expect(acc.GetLastOperation()).To(Equal(&gardencorev1beta1.LastOperation{Description: desc}))
				})
			})

			Describe("#GetLastError", func() {
				It("should get the last error", func() {
					var (
						desc = "desc"
						acc  = mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{LastError: &gardencorev1beta1.LastError{Description: "desc"}})
					)

					Expect(acc.GetLastError()).To(Equal(&gardencorev1beta1.LastError{Description: desc}))
				})
			})

			Describe("#GetConditions", func() {
				It("should get the conditions", func() {
					var (
						conditions = []gardencorev1beta1.Condition{
							{
								Type:           "ABC",
								Status:         gardencorev1beta1.ConditionTrue,
								Reason:         "reason",
								Message:        "message",
								LastUpdateTime: metav1.NewTime(metav1.Now().Round(time.Second)),
							},
						}
						acc = mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{Conditions: conditions})
					)
					getConditions := acc.GetConditions()
					Expect(getConditions).To(Equal(conditions))
				})
			})

			Describe("#GetState", func() {
				It("should get the extensions state", func() {
					state := &runtime.RawExtension{Raw: []byte("{\"raw\":\"ext\"}")}
					acc := mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{State: state})
					Expect(acc.GetState()).To(Equal(state))
				})
			})

			Describe("#GetResources", func() {
				It("should get the resources", func() {
					var (
						resources = []gardencorev1alpha1.NamedResourceReference{
							{
								Name: "test",
								ResourceRef: autoscalingv1.CrossVersionObjectReference{
									Kind:       "Secret",
									Name:       "test-secret",
									APIVersion: "v1",
								},
							},
						}
						acc = mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{Resources: resources})
					)
					getResources := acc.GetResources()
					Expect(getResources).To(Equal(resources))
				})
			})

			Describe("#SetConditions", func() {
				It("should set the conditions", func() {
					var (
						acc        = mkUnstructuredAccessorWithStatus(extensionsv1alpha1.DefaultStatus{})
						conditions = []gardencorev1beta1.Condition{
							{
								Type:           "ABC",
								Status:         gardencorev1beta1.ConditionTrue,
								Reason:         "reason",
								Message:        "message",
								LastUpdateTime: metav1.NewTime(metav1.Now().Round(time.Second)),
							},
						}
					)
					acc.SetConditions(conditions)
					getConditions := acc.GetConditions()
					Expect(getConditions).To(Equal(conditions))
				})
			})
		})
	})
})
