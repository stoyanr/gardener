// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
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

package botanist

import (
	"context"
	"fmt"

	"github.com/gardener/gardener/pkg/utils"
)

// DeployReferencedResources reads all referenced resources from the Garden cluster and writes them to the Seed cluster.
func (b *Botanist) DeployReferencedResources(ctx context.Context) error {
	for _, resourceRef := range b.Shoot.ResourceRefs {
		// Read the resource from the Garden cluster
		obj, err := utils.GetObjectByRef(ctx, b.K8sGardenClient.Client(), &resourceRef, b.Shoot.Info.Namespace)
		if err != nil {
			return err
		}
		if obj == nil {
			return fmt.Errorf("object not found %v", resourceRef)
		}

		// Write the resource to the Seed cluster
		if err := utils.CreateOrUpdateObjectByRef(ctx, b.K8sSeedClient.Client(), &resourceRef, b.Shoot.SeedNamespace, obj); err != nil {
			return err
		}
	}
	return nil
}

// DestroyReferencedResources deletes all referenced resources from the Seed cluster.
func (b *Botanist) DestroyReferencedResources(ctx context.Context) error {
	for _, resourceRef := range b.Shoot.ResourceRefs {
		// Delete the resource from the Seed cluster
		if err := utils.DeleteObjectByRef(ctx, b.K8sSeedClient.Client(), &resourceRef, b.Shoot.SeedNamespace); err != nil {
			return err
		}
	}
	return nil
}
