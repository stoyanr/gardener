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

package common

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

type Watchdog interface {
	Execute(ctx context.Context, f func(ctx context.Context) (reconcile.Result, error)) (reconcile.Result, error)
	Start(ctx context.Context) (context.Context, context.CancelFunc)
}

type watchdog struct {
	logger        logr.Logger
	recordToCheck string
	expected      string
}

func NewWatchdog(logger logr.Logger, recordToCheck, expected string) Watchdog {
	return &watchdog{
		logger:        logger,
		recordToCheck: recordToCheck,
		expected:      expected,
	}
}

func (w *watchdog) Start(ctx context.Context) (context.Context, context.CancelFunc) {
	watchdogCtx, watchdogCancelFunc := context.WithCancel(ctx)
	go func() {
		for {
			if w.leaderCheck(watchdogCtx) {
				watchdogCancelFunc()
				return
			}
			select {
			case <-time.After(2 * time.Minute):
			case <-watchdogCtx.Done():
				return
			}
		}
	}()

	return watchdogCtx, watchdogCancelFunc
}

func (w *watchdog) Execute(ctx context.Context, f func(ctx context.Context) (reconcile.Result, error)) (reconcile.Result, error) {
	watchdogCtx, watchdogCancelFunc := context.WithCancel(ctx)
	defer watchdogCancelFunc()

	go func() {
		for {
			if w.leaderCheck(watchdogCtx) {
				watchdogCancelFunc()
				return
			}
			select {
			case <-time.After(2 * time.Minute):
			case <-watchdogCtx.Done():
				return
			}
		}
	}()

	return f(ctx)
}

func (w *watchdog) leaderCheck(ctx context.Context) bool {
	owner, err := net.LookupTXT(fmt.Sprintf("owner.%s", w.recordToCheck))
	if err != nil {
		w.logger.Error(fmt.Errorf("Could not resolve owner DNS TXT record: %v", err), "namespace")
	}
	return owner[0] == string(w.expected)
}
