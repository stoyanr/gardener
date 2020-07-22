// Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package dns_test

import (
	"context"
	"fmt"

	cr "github.com/gardener/gardener/pkg/chartrenderer"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	"github.com/gardener/gardener/pkg/operation/botanist/component"
	. "github.com/gardener/gardener/pkg/operation/botanist/extensions/dns"
	. "github.com/gardener/gardener/test/gomega"

	dnsv1alpha1 "github.com/gardener/external-dns-management/pkg/apis/dns/v1alpha1"
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/version"
	"k8s.io/utils/pointer"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
)

var _ = Describe("#DNSOwner", func() {
	const (
		deployNS     = "test-chart-namespace"
		secretName   = "extensions-dns-test-deploy"
		dnsOwnerName = "test-deploy"
		ownerID      = "owner-id"
	)

	var (
		ctrl             *gomock.Controller
		ca               kubernetes.ChartApplier
		ctx              context.Context
		c                client.Client
		expectedDNSOwner *dnsv1alpha1.DNSOwner
		vals             *OwnerValues
		log              *logrus.Entry
		defaultDepWaiter component.DeployWaiter
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())

		ctx = context.TODO()
		log = logrus.NewEntry(logger.NewNopLogger())

		s := runtime.NewScheme()
		Expect(corev1.AddToScheme(s)).NotTo(HaveOccurred())
		Expect(dnsv1alpha1.AddToScheme(s)).NotTo(HaveOccurred())

		c = fake.NewFakeClientWithScheme(s)

		vals = &OwnerValues{
			Name:    "test-deploy",
			Active:  true,
			OwnerID: ownerID,
		}

		expectedDNSOwner = &dnsv1alpha1.DNSOwner{
			ObjectMeta: metav1.ObjectMeta{
				Name:      dnsOwnerName,
				Namespace: deployNS,
			},
			Spec: dnsv1alpha1.DNSOwnerSpec{
				OwnerId: ownerID,
				Active:  pointer.BoolPtr(true),
			},
		}

		ca = kubernetes.NewChartApplier(cr.NewWithServerVersion(&version.Info{}), kubernetes.NewApplier(c, meta.NewDefaultRESTMapper([]schema.GroupVersion{})))
		Expect(ca).NotTo(BeNil(), "should return chart applier")

		defaultDepWaiter = NewDNSOwner(vals, deployNS, ca, chartsRoot(), log, c)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#Deploy", func() {
		It("should create correct DNSOwner", func() {
			Expect(defaultDepWaiter.Deploy(ctx)).ToNot(HaveOccurred())

			actualDNSOwner := &dnsv1alpha1.DNSOwner{}
			err := c.Get(ctx, client.ObjectKey{Name: dnsOwnerName, Namespace: deployNS}, actualDNSOwner)

			Expect(err).NotTo(HaveOccurred())
			Expect(actualDNSOwner).To(DeepDerivativeEqual(expectedDNSOwner))
		})
	})
	Describe("#Destroy", func() {
		It("should not return error when it's not found", func() {
			Expect(defaultDepWaiter.Destroy(ctx)).ToNot(HaveOccurred())
		})

		It("should not return error when it's deleted successfully", func() {
			Expect(c.Create(ctx, expectedDNSOwner)).ToNot(HaveOccurred(), "adding pre-existing entry succeeds")

			Expect(defaultDepWaiter.Destroy(ctx)).ToNot(HaveOccurred())
		})

		It("should return err when fails to delete", func() {
			mc := mockclient.NewMockClient(ctrl)
			mc.EXPECT().Delete(ctx, &dnsv1alpha1.DNSOwner{
				ObjectMeta: metav1.ObjectMeta{
					Name:      dnsOwnerName,
					Namespace: deployNS,
				}}).Times(1).Return(fmt.Errorf("some random error"))

			Expect(NewDNSOwner(vals, deployNS, ca, chartsRoot(), log, mc).Destroy(ctx)).To(HaveOccurred())
		})
	})

	Describe("#WaitCleanup", func() {
		It("should not return error when it's already removed", func() {
			Expect(defaultDepWaiter.WaitCleanup(ctx)).ToNot(HaveOccurred())
		})
	})

})
