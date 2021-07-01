// Copyright (c) 2021 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	. "github.com/gardener/gardener/pkg/apis/extensions/validation"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/validation/field"
	"k8s.io/utils/pointer"
)

var _ = Describe("DNSRecord validation tests", func() {
	var dns *extensionsv1alpha1.DNSRecord

	BeforeEach(func() {
		dns = &extensionsv1alpha1.DNSRecord{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-dns",
				Namespace: "test-namespace",
			},
			Spec: extensionsv1alpha1.DNSRecordSpec{
				DefaultSpec: extensionsv1alpha1.DefaultSpec{
					Type: "provider",
				},
				SecretRef: corev1.SecretReference{
					Name: "test",
				},
				Name:       "test.example.com",
				RecordType: extensionsv1alpha1.DNSRecordTypeA,
				Values:     []string{"1.2.3.4"},
			},
		}
	})

	Describe("#ValidateDNSRecord", func() {
		It("should forbid empty DNSRecord resources", func() {
			errorList := ValidateDNSRecord(&extensionsv1alpha1.DNSRecord{})

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("metadata.namespace"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.type"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.secretRef.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.recordType"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeRequired),
				"Field": Equal("spec.values"),
			}))))
		})

		It("should forbid non-nil but empty region or zone", func() {
			dns.Spec.Region = pointer.StringPtr("")
			dns.Spec.Zone = pointer.StringPtr("")

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.region"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.zone"),
			}))))
		})

		It("should forbid name that is not a valid FQDN", func() {
			dns.Spec.Name = "test"

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.name"),
			}))))
		})

		It("should forbid unsupported recordType values", func() {
			dns.Spec.RecordType = "AAAA"

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeNotSupported),
				"Field": Equal("spec.recordType"),
			}))))
		})

		It("should forbid type CNAME and more than 1 value", func() {
			dns.Spec.RecordType = extensionsv1alpha1.DNSRecordTypeCNAME
			dns.Spec.Values = []string{"example.com", "foo.bar"}

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.values"),
			}))))
		})

		It("should forbid type A and a value that is not a valid IPv4 address", func() {
			dns.Spec.Values = []string{"example.com"}

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.values"),
			}))))
		})

		It("should forbid type CNAME and a value that is not a valid FQDN", func() {
			dns.Spec.RecordType = extensionsv1alpha1.DNSRecordTypeCNAME
			dns.Spec.Values = []string{"example"}

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.values"),
			}))))
		})

		It("should forbid negative ttl", func() {
			dns.Spec.TTL = pointer.Int64Ptr(-1)

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.ttl"),
			}))))
		})

		It("should allow valid resources (type A)", func() {
			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(BeEmpty())
		})

		It("should allow valid resources (type CNAME)", func() {
			dns.Spec.RecordType = extensionsv1alpha1.DNSRecordTypeCNAME
			dns.Spec.Values = []string{"example.com"}

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(BeEmpty())
		})

		It("should allow valid resources (type TXT)", func() {
			dns.Spec.RecordType = extensionsv1alpha1.DNSRecordTypeTXT
			dns.Spec.Values = []string{"can be anything"}

			errorList := ValidateDNSRecord(dns)

			Expect(errorList).To(BeEmpty())
		})
	})

	Describe("#ValidateDNSRecordUpdate", func() {
		It("should prevent updating anything if deletion time stamp is set", func() {
			now := metav1.Now()
			dns.DeletionTimestamp = &now
			newDNSRecord := prepareDNSRecordForUpdate(dns)
			newDNSRecord.DeletionTimestamp = &now
			newDNSRecord.Spec.SecretRef.Name = "changed-secretref-name"

			errorList := ValidateDNSRecordUpdate(newDNSRecord, dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec"),
			}))))
		})

		It("should prevent updating the type, name, or recordType", func() {
			newDNSRecord := prepareDNSRecordForUpdate(dns)
			newDNSRecord.Spec.Type = "changed-type"
			newDNSRecord.Spec.Name = "changed-test.example.com"
			newDNSRecord.Spec.RecordType = extensionsv1alpha1.DNSRecordTypeCNAME

			errorList := ValidateDNSRecordUpdate(newDNSRecord, dns)

			Expect(errorList).To(ConsistOf(PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.type"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.name"),
			})), PointTo(MatchFields(IgnoreExtras, Fields{
				"Type":  Equal(field.ErrorTypeInvalid),
				"Field": Equal("spec.recordType"),
			}))))
		})

		It("should allow updating everything else", func() {
			newDNSRecord := prepareDNSRecordForUpdate(dns)
			newDNSRecord.Spec.SecretRef.Name = "changed-secretref-name"
			newDNSRecord.Spec.Region = pointer.StringPtr("region")
			newDNSRecord.Spec.Zone = pointer.StringPtr("zone")
			newDNSRecord.Spec.Values = []string{"5.6.7.8"}
			newDNSRecord.Spec.TTL = pointer.Int64Ptr(300)

			errorList := ValidateDNSRecordUpdate(newDNSRecord, dns)

			Expect(errorList).To(BeEmpty())
		})
	})
})

func prepareDNSRecordForUpdate(obj *extensionsv1alpha1.DNSRecord) *extensionsv1alpha1.DNSRecord {
	newObj := obj.DeepCopy()
	newObj.ResourceVersion = "1"
	return newObj
}
