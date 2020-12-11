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

package managedseed

import (
	"time"

	"k8s.io/client-go/tools/cache"

	v1beta1constants "github.com/gardener/gardener/pkg/apis/core/v1beta1/constants"
	seedmanagementv1alpha1 "github.com/gardener/gardener/pkg/apis/seedmanagement/v1alpha1"
	"github.com/gardener/gardener/pkg/logger"
)

func (c *Controller) managedSeedAdd(obj interface{}, immediately bool) {
	key, err := cache.MetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	namespace, _, err := cache.SplitMetaNamespaceKey(key)
	if err != nil {
		return
	}
	if namespace != v1beta1constants.GardenNamespace {
		return
	}

	if immediately {
		logger.Logger.Debugf("Added ManagedSeed %s without delay to the queue", key)
		c.managedSeedQueue.AddAfter(key, 1*time.Second)
	} else {
		// Spread managedSeedistration of shooted seeds (including gardenlet updates/rollouts) across the configured sync jitter
		// period to avoid overloading the gardener-apiserver if all gardenlets in all shooted seeds are (re)starting
		// roughly at the same time
		// TODO Disabled for testing
		// duration := utils.RandomDurationWithMetaDuration(c.config.Controllers.ShootedSeedRegistration.SyncJitterPeriod) // TODO Add controller config
		logger.Logger.Infof("Added ManagedSeed %s with delay %s to the queue", key, 10*time.Second)
		c.managedSeedQueue.AddAfter(key, 10*time.Second)
	}
}

func (c *Controller) managedSeedUpdate(_, newObj interface{}) {
	newManagedSeed, ok := newObj.(*seedmanagementv1alpha1.ManagedSeed)
	if !ok {
		return
	}
	if newManagedSeed.Generation == newManagedSeed.Status.ObservedGeneration {
		return
	}
	c.managedSeedAdd(newObj, true)
}

func (c *Controller) managedSeedDelete(obj interface{}) {
	key, err := cache.DeletionHandlingMetaNamespaceKeyFunc(obj)
	if err != nil {
		return
	}
	c.managedSeedQueue.Add(key)
}