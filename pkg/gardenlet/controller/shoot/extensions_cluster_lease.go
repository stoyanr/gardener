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

package shoot

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gardener/gardener/pkg/apis/core/v1beta1"
	"github.com/gardener/gardener/pkg/controllerutils"
	"github.com/gardener/gardener/pkg/gardenlet/apis/config"
	"github.com/gardener/gardener/pkg/logger"
	gutil "github.com/gardener/gardener/pkg/utils/gardener"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/cache"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
	gardencorelisters "github.com/gardener/gardener/pkg/client/core/listers/core/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes/clientmap/keys"
	"github.com/gardener/gardener/pkg/extensions"
	shootpkg "github.com/gardener/gardener/pkg/operation/shoot"
)

func (c *Controller) extensionsClusterLeaseAdd(obj interface{}) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	logger.Logger.Info("Adding shoot to lease queue")
	c.extensionsClusterLeaseQueue.Add(key)
}

func (c *Controller) reconcileExtensionClusterLeaseControl(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	shoot, err := c.shootLister.Shoots(req.Namespace).Get(req.Name)
	if err != nil {
		if apierrors.IsNotFound(err) {
			logger.Logger.Debugf("[EXTENSION LEASE] Skipping Shoot %s because it has been deleted", req.NamespacedName)
			return reconcile.Result{}, nil
		}
		logger.Logger.Errorf("[EXTENSION LEASE] Could not get Shoot %s from store: %+v", req.NamespacedName, err)
		return reconcile.Result{}, err
	}

	gardenClient, err := c.clientMap.GetClient(ctx, keys.ForGarden())
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get garden client: %w", err)
	}

	project, err := gutil.ProjectForNamespaceFromReader(ctx, gardenClient.Client(), shoot.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	seed, err := c.k8sGardenCoreInformers.Core().V1beta1().Seeds().Lister().Get(*shoot.Spec.SeedName)
	if err != nil {
		return reconcile.Result{}, err
	}

	seedClient, err := c.clientMap.GetClient(ctx, keys.ForSeed(seed))
	if err != nil {
		return reconcile.Result{}, fmt.Errorf("failed to get client for seed %s: %w", seed.Name, err)
	}

	if err := c.extensionsClusterLeaseControl.Sync(ctx, seedClient.Client(), project.Name, shoot); err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{RequeueAfter: extensions.ClusterLeaseExpirationTimeout}, nil
}

type ExtensionClusterLeaseController interface {
	Sync(ctx context.Context, seedClient client.Client, projectName string, shoot *v1beta1.Shoot) error
}

type extensionsClusterLeaseController struct {
	nowFunc    func() time.Time
	config     *config.GardenletConfiguration
	seedLister gardencorelisters.SeedLister
}

// NewExtensionsClusterLeaseController constructs and returns a controller.
func NewExtensionsClusterLeaseController(nowFunc func() time.Time, config *config.GardenletConfiguration, seedLister gardencorelisters.SeedLister) ExtensionClusterLeaseController {
	return &extensionsClusterLeaseController{
		nowFunc:    nowFunc,
		config:     config,
		seedLister: seedLister,
	}
}

// Sync updates the ExtensionLease expiration timestamp in the cluster resource
func (c *extensionsClusterLeaseController) Sync(ctx context.Context, seedClient client.Client, projectName string, shoot *v1beta1.Shoot) error {
	if controllerutils.ShootIsManagedByThisGardenlet(shoot, c.config, c.seedLister) {
		cluster := &extensionsv1alpha1.Cluster{
			ObjectMeta: metav1.ObjectMeta{
				Name: shootpkg.ComputeTechnicalID(projectName, shoot),
			},
		}
		time, err := json.Marshal(metav1.NewMicroTime(c.nowFunc().UTC().Add(extensions.ClusterLeaseExpirationTimeout)))
		if err != nil {
			return err
		}
		patch := []byte(fmt.Sprintf(`{"spec":{"leaseExpiration":%s}}`, time))
		logger.Logger.Infof("Patching cluster resource %s with %s", cluster.Name, patch)
		return seedClient.Patch(ctx, cluster, client.RawPatch(types.MergePatchType, patch))
	}
	return nil
}
