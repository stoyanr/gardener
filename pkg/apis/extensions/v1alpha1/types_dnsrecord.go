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

package v1alpha1

import (
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var _ Object = (*DNSRecord)(nil)

// DNSRecordResource is a constant for the name of the DNSRecord resource.
const DNSRecordResource = "DNSRecord"

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object
// +kubebuilder:resource:scope=Namespaced,path=DNSRecords,shortName=cp,singular=DNSRecord
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name=Type,JSONPath=".spec.type",type=string,description="The control plane type."
// +kubebuilder:printcolumn:name=Purpose,JSONPath=".spec.purpose",type=string,description="Purpose of control plane resource."
// +kubebuilder:printcolumn:name=Status,JSONPath=".status.lastOperation.state",type=string,description="Status of control plane resource."
// +kubebuilder:printcolumn:name=Age,JSONPath=".metadata.creationTimestamp",type=date,description="creation timestamp"

// DNSRecord is a specification for a DNSRecord resource.
type DNSRecord struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec DNSRecordSpec `json:"spec"`
	// +optional
	Status DNSRecordStatus `json:"status"`
}

// GetExtensionSpec implements Object.
func (i *DNSRecord) GetExtensionSpec() Spec {
	return &i.Spec
}

// GetExtensionStatus implements Object.
func (i *DNSRecord) GetExtensionStatus() Status {
	return &i.Status
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// DNSRecordList is a list of DNSRecord resources.
type DNSRecordList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata"`

	// Items is the list of DNSRecords.
	Items []DNSRecord `json:"items"`
}

// DNSRecordSpec is the spec of a DNSRecord resource.
type DNSRecordSpec struct {
	// DefaultSpec is a structure containing common fields used by all extension resources.
	DefaultSpec `json:",inline"`
	// SecretRef is a reference to a secret that contains the cloud provider specific credentials.
	SecretRef corev1.SecretReference `json:"secretRef"`
	// Region is the region of this DNS record. If not specified, the region specified in SecretRef will be used.
	// If that is also not specified, a certain default region will be used, e.g. us-west-2 for AWS route53.
	// +optional
	Region *string `json:"region,omitempty"`
	// Zone is the DNS hosted zone of this DNS record. If not specified, it will be determined automatically by
	// getting all hosted zones of the account and searching for the longest zone name that is a suffix of Name.
	// +optional
	Zone *string `json:"zone,omitempty"`
	// Name is the fully qualified domain name, e.g. api.<shoot domain>.
	Name string `json:"name"`
	// RecordType is the DNS record type. Only A and TXT records are currently supported. Defaults to A.
	// +optional
	RecordType *DNSRecordType `json:"recordType,omitempty"`
	// Values is a list of IP addresses for A records, or texts for TXT records.
	Values []string `json:"values"`
	// TTL is the time to live in seconds. Defaults to 120.
	// +optional
	TTL *int64 `json:"ttl,omitempty"`
}

// DNSRecordStatus is the status of a DNSRecord resource.
type DNSRecordStatus struct {
	// DefaultStatus is a structure containing common fields used by all extension resources.
	DefaultStatus `json:",inline"`
	// Zone is the DNS hosted zone of this DNS record.
	// +optional
	Zone *string `json:"zone,omitempty"`
}

// DNSRecordType is a string alias.
type DNSRecordType string

const (
	// DNSRecordTypeA specifies that the DNSRecord is of type A.
	DNSRecordTypeA DNSRecordType = "A"
	// DNSRecordTypeTXT specifies that the DNSRecord is of type TXT.
	DNSRecordTypeTXT DNSRecordType = "TXT"
)
