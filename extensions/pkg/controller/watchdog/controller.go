/*
 * Copyright 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
 *
 *  Licensed under the Apache License, Version 2.0 (the "License");
 *  you may not use this file except in compliance with the License.
 *  You may obtain a copy of the License at
 *
 *       http://www.apache.org/licenses/LICENSE-2.0
 *
 *  Unless required by applicable law or agreed to in writing, software
 *  distributed under the License is distributed on an "AS IS" BASIS,
 *  WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  See the License for the specific language governing permissions and
 *  limitations under the License.
 *
 */

package watchdog

import (
	"context"
	"time"

	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	extensionscontroller "github.com/gardener/gardener/extensions/pkg/controller"
	extensionsv1alpha1 "github.com/gardener/gardener/pkg/apis/extensions/v1alpha1"
)

type AddArgs struct {
	ControllerOptions controller.Options
	// Predicates are the predicates to use.
	// If unset, GenerationChangedPredicate will be used.
	Predicates []predicate.Predicate
}

func Add(mgr manager.Manager, args AddArgs) error {
	args.ControllerOptions.Reconciler = NewReconciler()
	ctrl, err := controller.New("ClusterLeaseWatchdog", mgr, args.ControllerOptions)
	if err != nil {
		return err
	}

	if err := ctrl.Watch(&source.Kind{Type: &extensionsv1alpha1.Cluster{}}, &handler.EnqueueRequestForObject{}, args.Predicates...); err != nil {
		return err
	}
	return nil
}

type clusterLeaseWatchdog struct {
	clustersToCheck map[string]context.CancelFunc
	client          client.Client
}

func NewReconciler() *clusterLeaseWatchdog {
	return &clusterLeaseWatchdog{}
}

func (w *clusterLeaseWatchdog) Reconcile(ctx context.Context, req reconcile.Request) (reconcile.Result, error) {
	cancelFunc := w.clustersToCheck[req.Namespace]
	cluster, err := extensionscontroller.GetCluster(ctx, w.client, req.Namespace)
	if err != nil {
		return reconcile.Result{}, err
	}

	leaseExpired := time.Now().UTC().After(cluster.LeaseExpiration.Time)
	if leaseExpired {
		cancelFunc()
		return reconcile.Result{}, nil
	}

	return reconcile.Result{}, nil
}

func (w *clusterLeaseWatchdog) Register(ctx context.Context, namespacedName types.NamespacedName) context.Context {
	newCtx, cancelFunc := context.WithCancel(ctx)
	w.clustersToCheck[namespacedName.Namespace] = cancelFunc
	return newCtx
}
