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

package validation_test

import (
	gardencorev1beta1 "github.com/gardener/gardener/pkg/apis/core/v1beta1"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/gardener/gardener/pkg/apis/extensions/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

var _ = Describe("Extension validation tests", func() {
	var ext *extensionsv1alpha1.Extension

	BeforeEach(func() {
		ext = &extensionsv1alpha1.Extension{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-ext",
				Namespace: "test-namespace",
			},
			Spec: extensionsv1alpha1.ExtensionSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type:           "provider",
					ProviderConfig: &runtime.RawExtension{},
				},
			},
		}
	})

	Describe("#ValidExtension", func() {
		It("should forbid empty Extension resources", func() {
			errorList := ValidateExtension(&extensionsv1alpha1.Extension{})

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.type"),
			}))))
		})

		It("should forbid extensions with invalid resources", func() {
			ext.Spec.Resources = []gardencorev1beta1.NamedResourceReference{
				{},
			}
			errorList := ValidateExtension(ext)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.resources[0].name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.resources[0].resourceRef.kind"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.resources[0].resourceRef.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.resources[0].resourceRef.apiVersion"),
			}))))
		})

		It("should allow valid ext resources", func() {
			errorList := ValidateExtension(ext)

			Expect(errorList).To(BeEmpty())
		})
	})

	Describe("#ValidExtensionUpdate", func() {
		It("should prevent updating anything if deletion time stamp is set", func() {
			now := metav1.Now()
			ext.DeletionTimestamp = &now

			newExtension := prepareExtensionForUpdate(ext)
			newExtension.DeletionTimestamp = &now
			newExtension.Spec.ProviderConfig = nil

			errorList := ValidateExtensionUpdate(newExtension, ext)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should prevent updating the type and region", func() {
			newExtension := prepareExtensionForUpdate(ext)
			newExtension.Spec.Type = "changed-type"

			errorList := ValidateExtensionUpdate(newExtension, ext)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.type"),
			}))))
		})

		It("should allow updating the provider config", func() {
			newExtension := prepareExtensionForUpdate(ext)
			newExtension.Spec.ProviderConfig = nil

			errorList := ValidateExtensionUpdate(newExtension, ext)

			Expect(errorList).To(BeEmpty())
		})
	})
})

func prepareExtensionForUpdate(obj *extensionsv1alpha1.Extension) *extensionsv1alpha1.Extension {
	newObj := obj.DeepCopy()
	newObj.ResourceVersion = "1"
	return newObj
}
